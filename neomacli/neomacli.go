// Package neomacli provides a CLI harness for running neoma-based API servers
// with graceful shutdown, signal handling, and customizable options.
package neomacli

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/MeGaNeKoS/neoma/casing"
	"github.com/MeGaNeKoS/neoma/core"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

var optionsKey contextKey = "neoma/cli/options"

var durationType = reflect.TypeFor[time.Duration]()

// Logger is the interface used by the CLI for printing status messages.
type Logger interface {
	Println(args ...any)
}

// CLI is the interface for a runnable command-line application.
type CLI interface {
	Run()
	Root() *cobra.Command
}

// Hooks allows registering callbacks that run when the server starts and stops.
type Hooks interface {
	OnStart(func())
	OnStop(func())
}

// Option is a functional option for configuring the CLI.
type Option func(*cliConfig)

type defaultLogger struct {
	w io.Writer
}

func (l *defaultLogger) Println(args ...any) {
	_, _ = fmt.Fprintln(l.w, args...)
}

type contextKey string

type option struct {
	name string
	typ  reflect.Type
	path []int
}

type cli[Options any] struct {
	root     *cobra.Command
	optInfo  []option
	onParsed func(Hooks, *Options)
	start    func()
	stop     func()
	logger   Logger
}

func (c *cli[Options]) Run() {
	var o Options

	existing := c.root.PersistentPreRun
	c.root.PersistentPreRun = func(cmd *cobra.Command, args []string) {
		v := reflect.ValueOf(&o).Elem()
		flags := c.root.PersistentFlags()
		for _, opt := range c.optInfo {
			f := v
			for _, i := range opt.path {
				if f.Kind() == reflect.Pointer {
					if f.IsNil() {
						f.Set(reflect.New(f.Type().Elem()))
					}
					f = f.Elem()
				}
				f = f.Field(i)
			}

			fv := getValueFromFlagOrEnv(flags, opt, opt.typ)
			f.Set(fv)
		}

		c.onParsed(c, &o)

		if existing != nil {
			existing(cmd, args)
		}

		// Set options in context, so custom commands can access it.
		cmd.SetContext(context.WithValue(cmd.Context(), optionsKey, &o))
	}

	_ = c.root.Execute()
}

func (c *cli[O]) Root() *cobra.Command {
	return c.root
}

func (c *cli[O]) OnStart(fn func()) {
	c.start = fn
}

func (c *cli[O]) OnStop(fn func()) {
	c.stop = fn
}

func (c *cli[O]) registerOption(flags *pflag.FlagSet, field reflect.StructField, currentPath []int, name, defaultValue string) error {
	fieldType := core.Deref(field.Type)

	c.optInfo = append(c.optInfo, option{name, field.Type, currentPath})

	switch fieldType.Kind() {
	case reflect.String:
		flags.StringP(name, field.Tag.Get("short"), defaultValue, field.Tag.Get("doc"))
	case reflect.Int, reflect.Int64:
		var def int64
		if defaultValue != "" {
			if fieldType == durationType {
				t, err := time.ParseDuration(defaultValue)
				if err != nil {
					return fmt.Errorf("failed to parse duration for field %s: %w", field.Name, err)
				}
				def = int64(t)
			} else {
				var err error
				def, err = strconv.ParseInt(defaultValue, 10, 64)
				if err != nil {
					return fmt.Errorf("failed to parse int for field %s: %w", field.Name, err)
				}
			}
		}
		if fieldType == durationType {
			flags.DurationP(name, field.Tag.Get("short"), time.Duration(def), field.Tag.Get("doc"))
		} else {
			flags.Int64P(name, field.Tag.Get("short"), def, field.Tag.Get("doc"))
		}
	case reflect.Bool:
		var def bool
		if defaultValue != "" {
			var err error
			def, err = strconv.ParseBool(defaultValue)
			if err != nil {
				return fmt.Errorf("failed to parse bool for field %q: %w", field.Name, err)
			}
		}
		flags.BoolP(name, field.Tag.Get("short"), def, field.Tag.Get("doc"))
	default:
		return fmt.Errorf("unsupported option type for field %q: %q", field.Name, field.Type.Kind().String())
	}

	return nil
}

func (c *cli[O]) setupOptions(t reflect.Type, path []int, prefix string) error {
	flags := c.root.PersistentFlags()
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)

		if !field.IsExported() {
			c.logger.Println("warning: ignoring unexported options field", field.Name)
			continue
		}

		currentPath := append([]int{}, path...)
		currentPath = append(currentPath, i)

		fieldType := core.Deref(field.Type)
		if field.Anonymous {
			if err := c.setupOptions(fieldType, currentPath, prefix); err != nil {
				return err
			}
			continue
		}

		name := field.Tag.Get("name")
		if name == "" {
			name = casing.Kebab(field.Name)
		}

		if prefix != "" {
			name = prefix + "." + name
		}

		envName := "SERVICE_" + casing.Snake(strings.ReplaceAll(name, ".", "_"), strings.ToUpper)
		defaultValue := field.Tag.Get("default")
		if v, ok := os.LookupEnv(envName); ok {
			defaultValue = v
		}

		switch fieldType.Kind() {
		case reflect.String, reflect.Int, reflect.Int64, reflect.Bool:
			if err := c.registerOption(flags, field, currentPath, name, defaultValue); err != nil {
				return fmt.Errorf("failed to register option %q: %w", field.Name, err)
			}
		case reflect.Struct:
			if err := c.setupOptions(fieldType, currentPath, name); err != nil {
				return fmt.Errorf("failed to setup options for field %q: %w", field.Name, err)
			}
		case reflect.Pointer:
			if fieldType.Kind() == reflect.Struct {
				if err := c.setupOptions(fieldType, currentPath, name); err != nil {
					return fmt.Errorf("failed to setup options for field %q: %w", field.Name, err)
				}
			} else {
				return fmt.Errorf("unsupported option type for field %q: pointer to %q", field.Name, fieldType.Kind().String())
			}
		default:
			return fmt.Errorf("unsupported option type for field %q: %q", field.Name, field.Type.Kind().String())
		}
	}

	return nil
}

type cliConfig struct {
	logger Logger
}

// New creates a new CLI instance that parses the given options type from flags
// and environment variables, then calls onParsed to set up the server.
func New[O any](onParsed func(Hooks, *O), opts ...Option) CLI {
	cfg := &cliConfig{
		logger: &defaultLogger{w: os.Stderr},
	}
	for _, opt := range opts {
		opt(cfg)
	}

	use := "app"
	if len(os.Args) > 0 {
		use = filepath.Base(os.Args[0])
	}

	c := &cli[O]{
		root: &cobra.Command{
			Use: use,
		},
		onParsed: onParsed,
		logger:   cfg.logger,
	}

	var o O
	if err := c.setupOptions(reflect.TypeOf(o), []int{}, ""); err != nil {
		panic(err)
	}

	c.root.Run = func(cmd *cobra.Command, args []string) {
		done := make(chan struct{}, 1)
		if c.start != nil {
			go func() {
				c.start()
				done <- struct{}{}
			}()
		}

		quit := make(chan os.Signal, 1)
		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

		select {
		case <-done:
		case <-quit:
			if c.stop != nil {
				c.logger.Println("Gracefully shutting down the server...")
				c.stop()
			}
		}
	}
	return c
}

// WithLogger returns an Option that sets the logger used by the CLI.
func WithLogger(l Logger) Option {
	return func(c *cliConfig) {
		c.logger = l
	}
}

// WithOptions returns a cobra run function that extracts the parsed options
// from the command context and passes them to f.
func WithOptions[Options any](f func(cmd *cobra.Command, args []string, options *Options)) func(*cobra.Command, []string) {
	return func(cmd *cobra.Command, s []string) {
		options, _ := cmd.Context().Value(optionsKey).(*Options)
		f(cmd, s, options)
	}
}

func getBoolValue(flags *pflag.FlagSet, flagName, envValue string, hasEnv bool) bool {
	if flags.Changed(flagName) {
		value, _ := flags.GetBool(flagName)
		return value
	} else if hasEnv {
		value, err := strconv.ParseBool(envValue)
		if err == nil {
			return value
		}
	}
	value, _ := flags.GetBool(flagName)
	return value
}

func getEnvName(flagName string) string {
	name := strings.ReplaceAll(flagName, "-", "_")
	name = strings.ReplaceAll(name, ".", "_")
	return "SERVICE_" + strings.ToUpper(name)
}

func getIntValue(flags *pflag.FlagSet, flagName, envValue string, hasEnv bool, isDuration bool) any {
	if flags.Changed(flagName) {
		if isDuration {
			value, _ := flags.GetDuration(flagName)
			return value
		}
		value, _ := flags.GetInt64(flagName)
		return value
	} else if hasEnv {
		if isDuration {
			value, err := time.ParseDuration(envValue)
			if err == nil {
				return value
			}
		} else {
			value, err := strconv.ParseInt(envValue, 10, 64)
			if err == nil {
				return value
			}
		}
	}

	if isDuration {
		value, _ := flags.GetDuration(flagName)
		return value
	}
	value, _ := flags.GetInt64(flagName)
	return value
}

func getStringValue(flags *pflag.FlagSet, flagName, envValue string, hasEnv bool) string {
	if flags.Changed(flagName) {
		value, _ := flags.GetString(flagName)
		return value
	} else if hasEnv {
		return envValue
	}
	value, _ := flags.GetString(flagName)
	return value
}

func getValueFromFlagOrEnv(flags *pflag.FlagSet, opt option, fieldType reflect.Type) reflect.Value {
	value, ok := getValueFromType(flags, opt.name, fieldType)
	if !ok {
		panic(fmt.Sprintf("unsupported type for option %s: %s", opt.name, fieldType.String()))
	}

	fv := reflect.ValueOf(value)
	if fieldType.Kind() == reflect.Pointer {
		ptr := reflect.New(fv.Type())
		ptr.Elem().Set(fv)
		fv = ptr
	}

	return fv
}

func getValueFromType(flags *pflag.FlagSet, flagName string, fieldType reflect.Type) (any, bool) {
	envName := getEnvName(flagName)
	envValue, hasEnv := os.LookupEnv(envName)

	switch core.Deref(fieldType).Kind() {
	case reflect.String:
		return getStringValue(flags, flagName, envValue, hasEnv), true
	case reflect.Int, reflect.Int64:
		isDuration := fieldType == durationType
		rawValue := getIntValue(flags, flagName, envValue, hasEnv, isDuration)
		return reflect.ValueOf(rawValue).Convert(core.Deref(fieldType)).Interface(), true
	case reflect.Bool:
		return getBoolValue(flags, flagName, envValue, hasEnv), true
	default:
		return nil, false
	}
}
