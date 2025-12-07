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

Some definitions of "types" in AWS-SDK do not work with API-defined
XML.  AWS-SDK has specific routines in marshaling for those types.

An example is "types.Tagging" used in the "PutObjectTagging" action.
The XML Marshaler of Golang stdlib produces an XML output lacking
"<Tag>" entry, which appears in the XML in the API document.  Looking
at the API document, Tagging's "<Tag>" entry has no html-link, i.e.,
no definition.  See the following description for the difference of
the generated XML.

This is the list of structure slots that require the same encoding.

- **Buckets []Bucket** slot used in the response of ListBuckets and
  ListDirectoryBuckets.
- **Grants []Grant** slot in types.AccessControlPolicy.
- **AccessControlList []Grant** slot in types.S3Location.
- **OptionalFields []InventoryOptionalField** slot in
  types.InventoryConfiguration.
- **RoutingRules []RoutingRule** slot used in GetBucketWebsite and
  PutBucketWebsite.
- **Tags []Tag** slot (in several structures).
- **TagSet []Tag** slot in types.Tagging.
- **TargetGrants []TargetGrant** slot in types.LoggingEnabled.
- **UserMetadata []MetadataEntry** slot in types.S3Location.

AWS-SDK has its own marshalers for such types.  "Tagging" has
"awsRestxml_serializeDocumentTagging()" in
"aws-sdk-go-v2/service/s3/serializers.go".
"awsRestxml_serializeDocumentTagging()" calls
"awsRestxml_serializeDocumentTagSet()" and
"awsRestxml_serializeDocumentTag()".

Thus, we need to prepare separate type definitions for marshaling.
They are in "auxiliary.go".  They are hand-coded because the ad-hoc
stub-generator is not cleaver enough to generate needed types from the
Smithy definition.

The Tagging type shall be marshaled in the API document as follows.

```
<Tagging>
  <TagSet>
    <Tag><Key>mytag1</Key><Value>myvalue1</Value></Tag>
    <Tag><Key>mytag2</Key><Value>myvalue2</Value></Tag>
    <Tag><Key>mytag3</Key><Value>myvalue3</Value></Tag>
  </TagSet>
</Tagging>
```

First, the definition of "types.Tag" is not a problem.

```
type Tag struct {
    Key *string
    Value *string
}
```

The definition of "types.Tagging" in AWS-SDK is as follows.

```
type Tagging struct {
    TagSet []Tag
}
```

By this definition, Golang's marshaler works on an XML like the
following.  Notice the <Tag> is missing that is not we expected.  This
is due to the fact that "Tag" does not appear as a name.

```
<Tagging>
  <TagSet><Key>mytag1</Key><Value>myvalue1</Value></TagSet>
  <TagSet><Key>mytag2</Key><Value>myvalue2</Value></TagSet>
  <TagSet><Key>mytag3</Key><Value>myvalue3</Value></TagSet>
</Tagging>
```

To get the wanted XML output, the type definitions should be modified
to the following.

```
type Tagging struct {
    TagSet TagSet
}
type Tagging struct {
    Tag []Tag
}
```

The extraction of the type definitions in Smithy is shown (details
dropped).  Notice the "xmlName:Tag" is attached on the "Tag" type in
the "TagSet" type definition.  It instructs "Tag" to appear.

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
