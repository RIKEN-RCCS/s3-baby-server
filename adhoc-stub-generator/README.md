# README.md

An ad-hoc stub-generator generates server-side stubs for AWS-S3 from
the API definition in Smithy.  It generates files: "api-template.go",
"handler.go", "dispatcher.go", and "marshaler.go".  These generated
files are copied in the source directory of Baby-server.

## About "s3.json", AWS-S3 Definition in Smithy IDL

This ad-hoc stub-generator refers to "s3.json" in Golang's
aws-sdk-go-v2.  There are several "s3.json" files in the world but
they are virtually identical (but not exactly identical).  For
instance, other files can be found in Smithy-rust or Smithy-java.

- https://github.com/aws/aws-sdk-go-v2/codegen/sdk-codegen/aws-models/s3.json
- https://github.com/smithy-lang/smithy-rs/blob/main/aws/sdk/aws-models/s3.json
- https://github.com/smithy-lang/smithy-java/blob/main/aws/client/aws-client-rulesengine/src/shared-resources/software/amazon/smithy/java/aws/client/rulesengine/s3.json

Note that AWS's Simity for Golang does not contain "s3.json".  See
smithy-go:

- https://github.com/aws/smithy-go

General information on Smithy IDL and its syntax can be found in:

- https://smithy.io
- https://smithy.io/2.0/spec/idl.html

## Notable remarks

## Request: API declaration vs SDK structure

XXXXRequest (API) -> XXXXInput

## Respone: API declaration vs SDK structure -- Re

XXXXResult (API) -> XXXXOutput

## Errors defined

Error types defined are: {"BucketAlreadyExists"
"BucketAlreadyOwnedByYou" "EncryptionTypeMismatch"
"IdempotencyParameterMismatch" "InvalidObjectState" "InvalidRequest"
"InvalidWriteOffset" "NoSuchBucket" "NoSuchKey" "NoSuchUpload"
"NotFound" "ObjectAlreadyInActiveTierError"
"ObjectNotInActiveTierError" "TooManyParts"}

Definitions of errors have no member slots.

## Note on Parameters in a s3.XXXXInput

Presence of a query/header in a request can be checked by nil for
primitive types (Boolean/Integer/Time) and by "" for enumerations,
when they are stored in s3.XXXXInput.  Note an empty string is
distinct from values of enumerations.

Use of "strconv.ParseBool" for booleans in queries and headers may be
sloppy.  What should be accepted for truth values.

## Input/Output Records

Handling input records is straightforward.  There are three cases of
handling output records.  Examples are following API actions.

- DeleteBucket : DeleteBucketRequest → Unit (in Smithy)
- CopyObject : CopyObjectRequest → CopyObjectOutput (in Smithy)
- ListBuckets : ListBucketsRequest → ListBucketsOutput (in Smithy)

"DeleteBucket" has "Unit" as an output type.  It means a returned
response contains nothing.

"CopyObjectOutput" in Smithy has "CopyObjectResult" member which is
mared by "httpPayload" (in traits).  It means that member is returned
as a pyload in a response.

"ListBucketsOutput" in Smithy is marked by "xmlName" with
"ListAllMyBucketsResult" (in traits).  It indicates an output record
itself is returned as a payload, but its name is replaced by a name
cited by "xmlName".

### An extra field in output records

Output records ("XXXXOutput") have an extra slot "ResultMetadata".  It
is not a pointer and the default xml-marshaler outputs the xml-tag
even if it is empty.  Some work is needed to drop the slot.

----------------------------------------------------------------

## MEMO: API Markers in Smithy

There are API Markers in traits in Smithy ("smithy.api#XXXX").
Entries marked by (+) are handled (in some way) in this
stub-generator.

- "default"
- "deprecated"
- "enumValue" (+)
- "eventPayload"
- "hostLabel"
- "httpHeader" (+)
- "httpLabel" (+)
- "httpPayload" (+)
- "httpPrefixHeaders" (+)
- "httpQuery" (+)
- "idempotencyToken"
- "required" (+)
- "xmlAttribute" (+)
- "xmlFlattened" (+)
- "xmlName" (+)
- "xmlNamespace"

## MEMO

There is an extra slot in AWS-SDK, in "XXXXOutput".
- ResultMetadata middleware.Metadata

## XML Tag Correction

There is no direct correspondence between type definitions and XML
rendering in the Smithy definition.  In other words, Smithy specifies
both the type definition and its way of XML rendering.  AWS-SDK has
specific marshaling routines for types, and we cannot use the Golang's
standard marshaler.

An example is "types.Tagging" used in the "PutObjectTagging" action.
The standard marshaler produces an XML output lacking `<Tag>` entry,
which appears in the XML in the API document.  Looking at the AWS-SDK
API document, the Tagging type has no corresponding definition to the
`<Tag>` entry.

See the following description for the difference of the generated XML.

### XML Missing Tag Insertion ("Tag-Affix")

The extraction shown below is the type definitions in Smithy (details
dropped).  Notice the "xmlName:Tag" is attached on the "Tag" type in
the "TagSet" type definition.  It instructs "Tag" to appear in place
of "TagSet" in XML.

```
"com.amazonaws.s3#Tagging": {
    "type": "structure",
    "members": {
        "TagSet": {
            "target": "com.amazonaws.s3#TagSet",
            }
        }
    }
},
"com.amazonaws.s3#TagSet": {
    "type": "list",
    "member": {
        "target": "com.amazonaws.s3#Tag",
        "traits": {
            "smithy.api#xmlName": "Tag"
        }
    }
},
"com.amazonaws.s3#Tag": {
    "type": "structure",
    "members": { ...... },
},
```

#### Specific Source of the Problem of Missing Tags

The type "Tagging" shall be marshaled in the AWS-S3 API document as
follows.

```
<Tagging>
  <TagSet>
    <Tag><Key>mytag1</Key><Value>myvalue1</Value></Tag>
    <Tag><Key>mytag2</Key><Value>myvalue2</Value></Tag>
    <Tag><Key>mytag3</Key><Value>myvalue3</Value></Tag>
  </TagSet>
</Tagging>
```

The definitions of "types.Tag" and "types.Tagging" in AWS-SDK are as
follows.  Note the definition of "types.Tag" is no problem.

```
type Tag struct {
    Key *string
    Value *string
}
type Tagging struct {
    TagSet []Tag
}
```

By this definition, the standard marshaler renders an XML like the
following.  Notice the expected `<Tag>` is missing.  This is due to
the fact that "Tag" does not appear as a slot name (only as a type
name that will not appear in the XML rendering).

```
<Tagging>
  <TagSet><Key>mytag1</Key><Value>myvalue1</Value></TagSet>
  <TagSet><Key>mytag2</Key><Value>myvalue2</Value></TagSet>
  <TagSet><Key>mytag3</Key><Value>myvalue3</Value></TagSet>
</Tagging>
```

To get the wanted XML output using the standard marshaler, the type
definitions should be modified to the following.

```
type Tagging struct {
    TagSet TagSet
}
type Tagging struct {
    Tag []Tag
}
```

#### Implementation Restrictions of Tag-Affix

Correction of XML tags by tag-affix works only on the top level slots
of records.  It does not work when correction is needed in nested
slots.  The records needed in Baby-server are "[]Bucket" and "[]Tag",
and both appear in shallow slots.

#### Types that need XML Tag Insertion

- **Buckets []Bucket** slot in the response of ListBuckets.
- **TagSet []Tag** slot in the request of PutObjectTagging.

#### Full List of Types with Missing Tags

This is the list of record slots that require the non-standard
marshaling in AWS-S3.

- **Buckets []Bucket** slot used in the response of ListBuckets and
  ListDirectoryBuckets.
- **Grants []Grant** slot in types.AccessControlPolicy.
- **AccessControlList []Grant** slot in types.S3Location.
- **OptionalFields []InventoryOptionalField** slot in
  types.InventoryConfiguration.
- **RoutingRules []RoutingRule** slot used in GetBucketWebsite and
  PutBucketWebsite.
- **Tags []Tag** slot (in several records).
- **TagSet []Tag** slot in types.Tagging.
- **TargetGrants []TargetGrant** slot in types.LoggingEnabled.
- **UserMetadata []MetadataEntry** slot in types.S3Location.

In AWS-SDK, all types have their own generated marshalers.  They affix
missing tags for those types above.  For example, "Tagging" has
"awsRestxml_serializeDocumentTagging()" in
"aws-sdk-go-v2/service/s3/serializers.go".
"awsRestxml_serializeDocumentTagging()" calls
"awsRestxml_serializeDocumentTagSet()" and
"awsRestxml_serializeDocumentTag()".

This stub-generator prepares separate type definitions which can work
with the standard marshaler.

### XML Tag Mapping ("Tag-Amend")

DeleteObjects action has "Delete" parameter whose definition in Smithy
is as follows.  It has "Objects" slot as an array.

```
"com.amazonaws.s3#Delete": {
    "type": "structure",
    "members": {
        "Objects": {
            "target": "com.amazonaws.s3#ObjectIdentifierList",
            "traits": {
                "smithy.api#xmlFlattened": {},
                "smithy.api#xmlName": "Object"
    ......
}
```

The type of "Delete" is "types.Delete" and it is defined as follows in
AWS-SDK (attached the package name explicitly).

```
type types.Delete struct {
    Objects []types.ObjectIdentifier
    Quiet *bool
}
```

By this definition, the standard marshaler would rendar the "Objects"
slot as a list of `<Objects>`, although API definition requires a list
of `<Object>`.

To instruct the marshaler of this fact, the definition of the
"Objects" slot in Smithy has markers `xmlName="Object"` and
`xmlFlattened` in its traits.  Since the standard marshaler handles
arrays as flattened by default, it requires only the correction of tag
names.

#### Types that need XML Tag Mapping

- types.CompletedMultipartUpload for CompleteMultipartUpload (in request)
- types.Delete for DeleteObjects (in request)
- types.Errors for DeleteObjects (in response)
- types.Uploads for ListMultipartUploads (in response)
- types.Parts for ListParts (in response)
