// Error Codes of AWS S3

// This file is derived from the copyright material by Amazon Web
// Services, Inc.
//
// This defines error codes AWS S3.  The code in this file is
// extracted from the AWS S3 API specification.  It contains a list of
// error codes in the sections "Error" and "Error responses".
//
// See
//   - https://docs.aws.amazon.com/AmazonS3/latest/API/API_Error.html
//
// Aws_s3_error_code is (an enumeration) of string type.
// Aws_s3_error_to_message is a map from error-code to a pair of an
// http status-code and a message.  Some of the messages that are
// rather long are shortened by hand.  Entries may have -1 for http
// status-code, which corresponds to "N/A" in the specification.

// Errors extending smithy.APIError are defined as listed (14 types).
//
//  - types.BucketAlreadyExists
//  - types.BucketAlreadyOwnedByYou
//  - types.EncryptionTypeMismatch
//  - types.IdempotencyParameterMismatch
//  - types.InvalidObjectState
//  - types.InvalidRequest
//  - types.InvalidWriteOffset
//  - types.NoSuchBucket
//  - types.NoSuchKey
//  - types.NoSuchUpload
//  - types.NotFound
//  - types.ObjectAlreadyInActiveTierError
//  - types.ObjectNotInActiveTierError
//  - types.TooManyParts

package server

import (
	"encoding/xml"
	"fmt"
	"net/http"

	smithy "github.com/aws/smithy-go"
)

// Elements of Errors.  It mimics a record described in "Error
// responses" section, but it is different from types.Error.  This
// implements smithy.APIError.  The Code slot is a string (not an
// enumeration) as it is copied to types.Error.  The headers slot is
// to set appropriate response headers.  It is only used on errors of
// NotModified and PreconditionFailed (although it is not required to
// return the specific headers in PreconditionFailed).  Note that
// attached xml-tag forces structures that extend this to be marshaled
// with the "Error" tag, too (although none extends Aws_s3_error,
// currently).
type Aws_s3_error struct {
	XMLName xml.Name `xml:"Error"`
	//Code Aws_s3_error_code
	Code      string
	Message   string
	Resource  string
	RequestId string
	headers   http.Header `json:"-"`
}

func (e *Aws_s3_error) Error() string {
	var code = e.ErrorCode()
	var m = e.ErrorMessage()
	if len(m) == 0 {
		return fmt.Sprintf("%s", code)
	} else {
		return fmt.Sprintf("%s: %s", code, m)
	}
}

func (e *Aws_s3_error) ErrorCode() string {
	return string(e.Code)
}

func (e *Aws_s3_error) ErrorMessage() string {
	return e.Message
}

func (e *Aws_s3_error) ErrorFault() smithy.ErrorFault {
	return smithy.FaultServer
}

type Aws_s3_error_code string

// Information on errors.  It is a pair of an http status-code and a
// message.
type Aws_s3_error_message struct {
	Status  int
	Message string
}

// [List of error codes]

const (
	AccessDenied                            = "AccessDenied"
	AccountProblem                          = "AccountProblem"
	AllAccessDisabled                       = "AllAccessDisabled"
	AmbiguousGrantByEmailAddress            = "AmbiguousGrantByEmailAddress"
	AuthorizationHeaderMalformed            = "AuthorizationHeaderMalformed"
	BadDigest                               = "BadDigest"
	BucketAlreadyExists                     = "BucketAlreadyExists"
	BucketAlreadyOwnedByYou                 = "BucketAlreadyOwnedByYou"
	BucketNotEmpty                          = "BucketNotEmpty"
	CredentialsNotSupported                 = "CredentialsNotSupported"
	CrossLocationLoggingProhibited          = "CrossLocationLoggingProhibited"
	EntityTooSmall                          = "EntityTooSmall"
	EntityTooLarge                          = "EntityTooLarge"
	ExpiredToken                            = "ExpiredToken"
	IllegalVersioningConfigurationException = "IllegalVersioningConfigurationException"
	IncompleteBody                          = "IncompleteBody"
	IncorrectNumberOfFilesInPostRequest     = "IncorrectNumberOfFilesInPostRequest"
	InlineDataTooLarge                      = "InlineDataTooLarge"
	InternalError                           = "InternalError"
	InvalidAccessKeyId                      = "InvalidAccessKeyId"
	InvalidAddressingHeader                 = "InvalidAddressingHeader"
	InvalidArgument                         = "InvalidArgument"
	InvalidBucketName                       = "InvalidBucketName"
	InvalidBucketState                      = "InvalidBucketState"
	InvalidDigest                           = "InvalidDigest"
	InvalidEncryptionAlgorithmError         = "InvalidEncryptionAlgorithmError"
	InvalidLocationConstraint               = "InvalidLocationConstraint"
	InvalidObjectState                      = "InvalidObjectState"
	InvalidPart                             = "InvalidPart"
	InvalidPartOrder                        = "InvalidPartOrder"
	InvalidPayer                            = "InvalidPayer"
	InvalidPolicyDocument                   = "InvalidPolicyDocument"
	InvalidRange                            = "InvalidRange"
	InvalidRequest                          = "InvalidRequest"
	InvalidSecurity                         = "InvalidSecurity"
	InvalidSOAPRequest                      = "InvalidSOAPRequest"
	InvalidStorageClass                     = "InvalidStorageClass"
	InvalidTargetBucketForLogging           = "InvalidTargetBucketForLogging"
	InvalidToken                            = "InvalidToken"
	InvalidURI                              = "InvalidURI"
	KeyTooLongError                         = "KeyTooLongError"
	MalformedACLError                       = "MalformedACLError"
	MalformedPOSTRequest                    = "MalformedPOSTRequest"
	MalformedXML                            = "MalformedXML"
	MaxMessageLengthExceeded                = "MaxMessageLengthExceeded"
	MaxPostPreDataLengthExceededError       = "MaxPostPreDataLengthExceededError"
	MetadataTooLarge                        = "MetadataTooLarge"
	MethodNotAllowed                        = "MethodNotAllowed"
	MissingAttachment                       = "MissingAttachment"
	MissingContentLength                    = "MissingContentLength"
	MissingRequestBodyError                 = "MissingRequestBodyError"
	MissingSecurityElement                  = "MissingSecurityElement"
	MissingSecurityHeader                   = "MissingSecurityHeader"
	NoLoggingStatusForKey                   = "NoLoggingStatusForKey"
	NoSuchBucket                            = "NoSuchBucket"
	NoSuchBucketPolicy                      = "NoSuchBucketPolicy"
	NoSuchKey                               = "NoSuchKey"
	NoSuchLifecycleConfiguration            = "NoSuchLifecycleConfiguration"
	NoSuchUpload                            = "NoSuchUpload"
	NoSuchVersion                           = "NoSuchVersion"
	NotImplemented                          = "NotImplemented"
	NotSignedUp                             = "NotSignedUp"
	OperationAborted                        = "OperationAborted"
	PermanentRedirect                       = "PermanentRedirect"
	PreconditionFailed                      = "PreconditionFailed"
	Redirect                                = "Redirect"
	RestoreAlreadyInProgress                = "RestoreAlreadyInProgress"
	RequestIsNotMultiPartContent            = "RequestIsNotMultiPartContent"
	RequestTimeout                          = "RequestTimeout"
	RequestTimeTooSkewed                    = "RequestTimeTooSkewed"
	RequestTorrentOfBucketError             = "RequestTorrentOfBucketError"
	SignatureDoesNotMatch                   = "SignatureDoesNotMatch"
	ServiceUnavailable                      = "ServiceUnavailable"
	SlowDown                                = "SlowDown"
	TemporaryRedirect                       = "TemporaryRedirect"
	TokenRefreshRequired                    = "TokenRefreshRequired"
	TooManyBuckets                          = "TooManyBuckets"
	UnexpectedContent                       = "UnexpectedContent"
	UnresolvableGrantByEmailAddress         = "UnresolvableGrantByEmailAddress"
	UserKeyMustBeSpecified                  = "UserKeyMustBeSpecified"
)

// [List of Tagging-related error codes]

const (
	// InvalidRequest = "InvalidRequest"
	InvalidTag         = "InvalidTag"
	NoSuchResource     = "NoSuchResource"
	TagPolicyException = "TagPolicyException"
	TooManyTags        = "TooManyTags"
)

// [Others needed by http]

const (
	NotModified = "NotModified"
)

var Aws_s3_error_to_message = map[string]Aws_s3_error_message{
	//
	// [List of error codes]
	//

	AccessDenied:                            {403, "Access Denied"},
	AccountProblem:                          {403, "There is a problem with your Amazon Web Services account."},
	AllAccessDisabled:                       {403, "All access to this Amazon S3 resource has been disabled."},
	AmbiguousGrantByEmailAddress:            {400, "The email address you provided is associated with more than one account."},
	AuthorizationHeaderMalformed:            {400, "The authorization header you provided is invalid."},
	BadDigest:                               {400, "The Content-MD5 you specified did not match what we received."},
	BucketAlreadyExists:                     {409, "The requested bucket name is not available. The bucket namespace is shared by all users of the system. Please select a different name and try again."},
	BucketAlreadyOwnedByYou:                 {409, "The bucket you tried to create already exists, and you own it."},
	BucketNotEmpty:                          {409, "The bucket you tried to delete is not empty."},
	CredentialsNotSupported:                 {400, "This request does not support credentials."},
	CrossLocationLoggingProhibited:          {403, "Cross-location logging not allowed. Buckets in one geographic location cannot log information to a bucket in another location."},
	EntityTooSmall:                          {400, "Your proposed upload is smaller than the minimum allowed object size."},
	EntityTooLarge:                          {400, "Your proposed upload exceeds the maximum allowed object size."},
	ExpiredToken:                            {400, "The provided token has expired."},
	IllegalVersioningConfigurationException: {400, "Indicates that the versioning configuration specified in the request is invalid."},
	IncompleteBody:                          {400, "You did not provide the number of bytes specified by the Content-Length HTTP header"},
	IncorrectNumberOfFilesInPostRequest:     {400, "POST requires exactly one file upload per request."},
	InlineDataTooLarge:                      {400, "Inline data exceeds the maximum allowed size."},
	InternalError:                           {500, "We encountered an internal error. Please try again."},
	InvalidAccessKeyId:                      {403, "The Amazon Web Services access key ID you provided does not exist in our records."},
	InvalidAddressingHeader:                 {-1, "You must specify the Anonymous role."},
	InvalidArgument:                         {400, "Invalid Argument"},
	InvalidBucketName:                       {400, "The specified bucket is not valid."},
	InvalidBucketState:                      {409, "The request is not valid with the current state of the bucket."},
	InvalidDigest:                           {400, "The Content-MD5 you specified is not valid."},
	InvalidEncryptionAlgorithmError:         {400, "The encryption request you specified is not valid. The valid value is AES256."},
	InvalidLocationConstraint:               {400, "The specified location constraint is not valid."},
	InvalidObjectState:                      {403, "The action is not valid for the current state of the object."},
	InvalidPart:                             {400, "One or more of the specified parts could not be found. The part might not have been uploaded, or the specified entity tag might not have matched the part's entity tag."},
	InvalidPartOrder:                        {400, "The list of parts was not in ascending order. Parts list must be specified in order by part number."},
	InvalidPayer:                            {403, "All access to this object has been disabled."},
	InvalidPolicyDocument:                   {400, "The content of the form does not meet the conditions specified in the policy document."},
	InvalidRange:                            {416, "The requested range cannot be satisfied."},
	InvalidRequest:                          {400, "Bad Request."},
	/*
		InvalidRequest: {400, "Please use AWS4-HMAC-SHA256."},
		InvalidRequest: {400, "SOAP requests must be made over an HTTPS connection."},
		InvalidRequest: {400, "Amazon S3 Transfer Acceleration is not supported for buckets with non-DNS compliant names."},
		InvalidRequest: {400, "Amazon S3 Transfer Acceleration is not supported for buckets with periods (.) in their names."},
		InvalidRequest: {400, "Amazon S3 Transfer Accelerate endpoint only supports virtual style requests."},
		InvalidRequest: {400, "Amazon S3 Transfer Accelerate is not configured on this bucket."},
		InvalidRequest: {400, "Amazon S3 Transfer Accelerate is disabled on this bucket."},
		InvalidRequest: {400, "Amazon S3 Transfer Acceleration is not supported on this bucket. Contact Amazon Web Services Support for more information."},
		InvalidRequest: {400, "Amazon S3 Transfer Acceleration cannot be enabled on this bucket. Contact Amazon Web Services Support for more information."},
	*/
	InvalidSecurity:                   {403, "The provided security credentials are not valid."},
	InvalidSOAPRequest:                {400, "The SOAP request body is invalid."},
	InvalidStorageClass:               {400, "The storage class you specified is not valid."},
	InvalidTargetBucketForLogging:     {400, "The target bucket for logging does not exist, is not owned by you, or does not have the appropriate grants for the log-delivery group."},
	InvalidToken:                      {400, "The provided token is malformed or otherwise invalid."},
	InvalidURI:                        {400, "Couldn't parse the specified URI."},
	KeyTooLongError:                   {400, "Your key is too long."},
	MalformedACLError:                 {400, "The XML you provided was not well-formed or did not validate against our published schema."},
	MalformedPOSTRequest:              {400, "The body of your POST request is not well-formed multipart/form-data."},
	MalformedXML:                      {400, "The XML you provided was not well-formed or did not validate against our published schema."},
	MaxMessageLengthExceeded:          {400, "Your request was too big."},
	MaxPostPreDataLengthExceededError: {400, "Your POST request fields preceding the upload file were too large."},
	MetadataTooLarge:                  {400, "Your metadata headers exceed the maximum allowed metadata size."},
	MethodNotAllowed:                  {405, "The specified method is not allowed against this resource."},
	MissingAttachment:                 {-1, "A SOAP attachment was expected, but none were found."},
	MissingContentLength:              {411, "You must provide the Content-Length HTTP header."},
	MissingRequestBodyError:           {400, "Request body is empty."},
	MissingSecurityElement:            {400, "The SOAP 1.1 request is missing a security element."},
	MissingSecurityHeader:             {400, "Your request is missing a required header."},
	NoLoggingStatusForKey:             {400, "There is no such thing as a logging status subresource for a key."},
	NoSuchBucket:                      {404, "The specified bucket does not exist."},
	NoSuchBucketPolicy:                {404, "The specified bucket does not have a bucket policy."},
	NoSuchKey:                         {404, "The specified key does not exist."},
	NoSuchLifecycleConfiguration:      {404, "The lifecycle configuration does not exist."},
	NoSuchUpload:                      {404, "The specified multipart upload does not exist. The upload ID might be invalid, or the multipart upload might have been aborted or completed."},
	NoSuchVersion:                     {404, "Indicates that the version ID specified in the request does not match an existing version."},
	NotImplemented:                    {501, "A header you provided implies functionality that is not implemented."},
	NotSignedUp:                       {403, "Your account is not signed up for the Amazon S3 service. You must sign up before you can use Amazon S3. You can sign up at the following URL:"},
	OperationAborted:                  {409, "A conflicting conditional action is currently in progress against this resource. Try again."},
	PermanentRedirect:                 {301, "The bucket you are attempting to access must be addressed using the specified endpoint. Send all future requests to this endpoint."},
	PreconditionFailed:                {412, "At least one of the preconditions you specified did not hold."},
	Redirect:                          {307, "Temporary redirect."},
	RestoreAlreadyInProgress:          {409, "Object restore is already in progress."},
	RequestIsNotMultiPartContent:      {400, "Bucket POST must be of the enclosure-type multipart/form-data."},
	RequestTimeout:                    {400, "Your socket connection to the server was not read from or written to within the timeout period."},
	RequestTimeTooSkewed:              {403, "The difference between the request time and the server's time is too large."},
	RequestTorrentOfBucketError:       {400, "Requesting the torrent file of a bucket is not permitted."},
	SignatureDoesNotMatch:             {403, "The request signature we calculated does not match the signature you provided.."},
	ServiceUnavailable:                {503, "Service is unable to handle request."},
	SlowDown:                          {503, "Reduce your request rate."},
	TemporaryRedirect:                 {307, "You are being redirected to the bucket while DNS updates."},
	TokenRefreshRequired:              {400, "The provided token must be refreshed."},
	TooManyBuckets:                    {400, "You have attempted to create more buckets than allowed."},
	UnexpectedContent:                 {400, "This request does not support content."},
	UnresolvableGrantByEmailAddress:   {400, "The email address you provided does not match any account on record."},
	UserKeyMustBeSpecified:            {400, "The bucket POST must contain the specified field name. If it is specified, check the order of the fields."},

	//
	// [List of Tagging-related error codes]
	//

	// InvalidRequest
	InvalidTag:         {400, "Tag key or value isn't valid."},
	NoSuchResource:     {404, "The specified resource doesn't exist."},
	TagPolicyException: {400, "The tag policy does not allow the specified value for the following tag key."},
	TooManyTags:        {400, "The number of tags exceeds the limit of 50 tags."},

	//
	// [Others needed by http]
	//

	NotModified: {304, "Not Modified."},
}
