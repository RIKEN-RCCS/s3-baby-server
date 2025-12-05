# README.md

## About "s3.json"

This ad hoc stub-generator refers to "s3.json" in Golang's
aws-sdk-go-v2.  There are many "s3.json" files but they are virtually
identical (but not exactly identical).  For instance, other "s3.json"
files can be found in Smithy-rust or Smithy-java.

https://github.com/aws/aws-sdk-go-v2/codegen/sdk-codegen/aws-models/s3.json

https://github.com/smithy-lang/smithy-rs/blob/main/aws/sdk/aws-models/s3.json

https://github.com/smithy-lang/smithy-java/blob/main/aws/client/aws-client-rulesengine/src/shared-resources/software/amazon/smithy/java/aws/client/rulesengine/s3.json

Note that AWS's Simity for Golang does not contain "s3.json".

https://github.com/aws/smithy-go

## Notable remarks

### Differing names in API and in structures

The definition in Smithy renames XML tags for structure slots.  An
example is "ListParts" action's "ListPartsOutput", where "Part" is in
the API and "Parts" is in the structure.  A structure slot is
specified by a member name, but an XML tag name is specified by
"smithy.api#xmlName".

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

## MEMO

There is an extra slot in AWS-SDK, in "XXXXOutput".
- ResultMetadata middleware.Metadata

## API Markers

There are API Markers in traits in Smithy ("smithy.api#XXXX").
Entries marked by (+) are handled (somewhat) in stub-generator.

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

## XML Marshaling/Unmarshaling

;; This part is related to AWS-SDK-defined types
;; {CompletedMultipartUpload, CreateBucketConfiguration, Delete,
;; Tagging}.

Some definitions of "types" in AWS-SDK does not generate API-defined
XML.  An example is "types.Tagging".  AWS-SDK has specific routines to
marshal/unmarshal for types.  See the following description for the
difference of the generated XML.

Thus, we prepared separate type definitions for the ad-hoc
stub-generator.  They are in "auxiliary.go".  They are hand-coded
because the ad-hoc stub-generator is not cleaver enough to generate
needed types from the Smithy definition.

Tagging type shall be marshaled in XML something like following.

```
<Tagging>
  <TagSet>
    <Tag><Key>mytag1</Key><Value>myvalue1</Value></Tag>
    <Tag><Key>mytag2</Key><Value>myvalue2</Value></Tag>
  </TagSet>
</Tagging>
```

First, the type Tag is defined as follows.  This is not a problem.

```
type Tag struct {
    Key *string
    Value *string
}
```

This is the definition of "types.Tagging" in AWS-SDK.

```
type Tagging struct {
    TagSet []Tag
}
```

For this definition, xml.Marshal() and xml.Unmarshal() works on an XML
like following which is not we expected.  Notice the <Tag> is missing.
This is due to the fact that `Tag` does not appear as a field name.

```
<Tagging>
  <TagSet>
    <Key>mytag1</Key><Value>myvalue1</Value>
    <Key>mytag2</Key><Value>myvalue2</Value>
  </TagSet>
</Tagging>
```

To get the wanted XML output, the definition should be modified to
something like following.

```
type Tagging struct {
    TagSet TagSet
}
type Tagging struct {
    Tag []Tag
}
```
