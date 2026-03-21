package core

// Validation error message templates. These package-level variables can be
// overridden to customize or localize the messages produced by schema
// validation. Templates containing %v or %s verbs are formatted with
// fmt.Sprintf at runtime.
var (
	// MsgUnexpectedProperty is used when an object contains a property not
	// defined in the schema.
	MsgUnexpectedProperty = "unexpected property"

	// MsgExpectedRFC3339DateTime is used when a string is not a valid
	// RFC 3339 date-time.
	MsgExpectedRFC3339DateTime = "expected string to be RFC 3339 date-time"

	// MsgExpectedRFC1123DateTime is used when a string is not a valid
	// RFC 1123 date-time.
	MsgExpectedRFC1123DateTime = "expected string to be RFC 1123 date-time"

	// MsgExpectedRFC3339Date is used when a string is not a valid RFC 3339
	// date.
	MsgExpectedRFC3339Date = "expected string to be RFC 3339 date"

	// MsgExpectedRFC3339Time is used when a string is not a valid RFC 3339
	// time.
	MsgExpectedRFC3339Time = "expected string to be RFC 3339 time"

	// MsgExpectedRFC5322Email is used when a string is not a valid RFC 5322
	// email address.
	MsgExpectedRFC5322Email = "expected string to be RFC 5322 email: %v"

	// MsgExpectedRFC5890Hostname is used when a string is not a valid
	// RFC 5890 hostname.
	MsgExpectedRFC5890Hostname = "expected string to be RFC 5890 hostname"

	// MsgExpectedRFC2673IPv4 is used when a string is not a valid RFC 2673
	// IPv4 address.
	MsgExpectedRFC2673IPv4 = "expected string to be RFC 2673 ipv4"

	// MsgExpectedRFC2373IPv6 is used when a string is not a valid RFC 2373
	// IPv6 address.
	MsgExpectedRFC2373IPv6 = "expected string to be RFC 2373 ipv6"

	// MsgExpectedRFCIPAddr is used when a string is not a valid IPv4 or
	// IPv6 address.
	MsgExpectedRFCIPAddr = "expected string to be either RFC 2673 ipv4 or RFC 2373 ipv6"

	// MsgExpectedRFC3986URI is used when a string is not a valid RFC 3986
	// URI.
	MsgExpectedRFC3986URI = "expected string to be RFC 3986 uri: %v"

	// MsgExpectedRFC4122UUID is used when a string is not a valid RFC 4122
	// UUID.
	MsgExpectedRFC4122UUID = "expected string to be RFC 4122 uuid: %v"

	// MsgExpectedRFC6570URITemplate is used when a string is not a valid
	// RFC 6570 URI template.
	MsgExpectedRFC6570URITemplate = "expected string to be RFC 6570 uri-template"

	// MsgExpectedRFC6901JSONPointer is used when a string is not a valid
	// RFC 6901 JSON pointer.
	MsgExpectedRFC6901JSONPointer = "expected string to be RFC 6901 json-pointer"

	// MsgExpectedRFC6901RelativeJSONPointer is used when a string is not a
	// valid RFC 6901 relative JSON pointer.
	MsgExpectedRFC6901RelativeJSONPointer = "expected string to be RFC 6901 relative-json-pointer"

	// MsgExpectedRegexp is used when a string is not a valid regular
	// expression.
	MsgExpectedRegexp = "expected string to be regex: %v"

	// MsgExpectedMatchAtLeastOneSchema is used when a value does not match
	// any schema in an anyOf constraint.
	MsgExpectedMatchAtLeastOneSchema = "expected value to match at least one schema but matched none"

	// MsgExpectedMatchExactlyOneSchema is used when a value does not match
	// exactly one schema in a oneOf constraint.
	MsgExpectedMatchExactlyOneSchema = "expected value to match exactly one schema but matched none"

	// MsgExpectedNotMatchSchema is used when a value matches a schema it
	// should not (not constraint).
	MsgExpectedNotMatchSchema = "expected value to not match schema"

	// MsgExpectedPropertyNameInObject is used when a propertyNames schema
	// value is not present in the object.
	MsgExpectedPropertyNameInObject = "expected propertyName value to be present in object"

	// MsgExpectedBoolean is used when a value is not a boolean.
	MsgExpectedBoolean = "expected boolean"

	// MsgExpectedDuration is used when a string is not a valid duration.
	MsgExpectedDuration = "expected duration: %v"

	// MsgExpectedNumber is used when a value is not a number.
	MsgExpectedNumber = "expected number"

	// MsgExpectedInteger is used when a value is not an integer.
	MsgExpectedInteger = "expected integer"

	// MsgExpectedString is used when a value is not a string.
	MsgExpectedString = "expected string"

	// MsgExpectedBase64String is used when a string is not valid base64.
	MsgExpectedBase64String = "expected string to be base64 encoded"

	// MsgExpectedArray is used when a value is not an array.
	MsgExpectedArray = "expected array"

	// MsgExpectedObject is used when a value is not an object.
	MsgExpectedObject = "expected object"

	// MsgExpectedArrayItemsUnique is used when array items are not unique.
	MsgExpectedArrayItemsUnique = "expected array items to be unique"

	// MsgExpectedOneOf is used when a value is not one of the allowed enum
	// values.
	MsgExpectedOneOf = "expected value to be one of %s"

	// MsgExpectedMinimumNumber is used when a number is below the minimum.
	MsgExpectedMinimumNumber = "expected number >= %v"

	// MsgExpectedExclusiveMinimumNumber is used when a number is at or
	// below the exclusive minimum.
	MsgExpectedExclusiveMinimumNumber = "expected number > %v"

	// MsgExpectedMaximumNumber is used when a number exceeds the maximum.
	MsgExpectedMaximumNumber = "expected number <= %v"

	// MsgExpectedExclusiveMaximumNumber is used when a number is at or
	// above the exclusive maximum.
	MsgExpectedExclusiveMaximumNumber = "expected number < %v"

	// MsgExpectedNumberBeMultipleOf is used when a number is not a multiple
	// of the required value.
	MsgExpectedNumberBeMultipleOf = "expected number to be a multiple of %v"

	// MsgExpectedMinLength is used when a string is shorter than the
	// minimum length.
	MsgExpectedMinLength = "expected length >= %d"

	// MsgExpectedMaxLength is used when a string exceeds the maximum
	// length.
	MsgExpectedMaxLength = "expected length <= %d"

	// MsgExpectedBePattern is used when a string does not match the
	// expected named pattern.
	MsgExpectedBePattern = "expected string to be %s"

	// MsgExpectedMatchPattern is used when a string does not match the
	// required regex pattern.
	MsgExpectedMatchPattern = "expected string to match pattern %s"

	// MsgExpectedMinItems is used when an array has fewer items than the
	// minimum.
	MsgExpectedMinItems = "expected array length >= %d"

	// MsgExpectedMaxItems is used when an array has more items than the
	// maximum.
	MsgExpectedMaxItems = "expected array length <= %d"

	// MsgExpectedMinProperties is used when an object has fewer properties
	// than the minimum.
	MsgExpectedMinProperties = "expected object with >= %d properties"

	// MsgExpectedMaxProperties is used when an object has more properties
	// than the maximum.
	MsgExpectedMaxProperties = "expected object with <= %d properties"

	// MsgExpectedRequiredProperty is used when a required property is
	// missing from the object.
	MsgExpectedRequiredProperty = "expected required property %s to be present"

	// MsgExpectedDependentRequiredProperty is used when a conditionally
	// required property is missing.
	MsgExpectedDependentRequiredProperty = "expected property %s to be present when %s is present"
)
