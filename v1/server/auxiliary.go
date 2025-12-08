// auxiliary.go

// API-STUB.  This file is part of API-STUB.

package server

import (
	"encoding/xml"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	//"io"
)

func xml_marshal_error(ty string, e error) error {
	var err1 = fmt.Errorf("Marshal error for type %s with %w", ty, e)
	var errz = &Aws_s3_error{Code: MalformedXML,
		Message: err1.Error()}
	return errz
}

// Marshaler of "Buckets []types.Bucket" slot.
//
// It is a slot in types.ListBucketsOutput.  The slot should be
// rendered as "<Buckets><Bucket>???</Bucket>...</Buckets>".

type H_Buckets struct {
	XMLName xml.Name `xml:"Buckets"`
	Bucket  []types.Bucket
}

func import_Buckets(d *xml.Decoder) ([]types.Bucket, error) {
	var o H_Buckets
	var err1 = d.Decode(&o)
	if err1 != nil {
		return nil, xml_marshal_error("[]Bucket", err1)
	}
	var i = o.Bucket
	return i, nil
}

func export_Buckets(e *xml.Encoder, i []types.Bucket) error {
	var o = H_Buckets{Bucket: i}
	var err1 = e.Encode(&o)
	if err1 != nil {
		return xml_marshal_error("[]Bucket", err1)
	}
	return nil
}

// Marshaler of "types.CreateBucketConfiguration"

type H_CreateBucketConfiguration struct {
	XMLName            xml.Name `xml:"CreateBucketConfiguration"`
	Bucket             *types.BucketInfo
	Location           *types.LocationInfo
	LocationConstraint types.BucketLocationConstraint
	Tags               struct {
		Tag []types.Tag
	}
}

/*
type H_Tags struct {
	XMLName xml.Name `xml:"Tags"`
	Tag     []types.Tag
}
*/

func import_CreateBucketConfiguration(d *xml.Decoder) (*types.CreateBucketConfiguration, error) {
	var o H_CreateBucketConfiguration
	var err1 = d.Decode(&o)
	if err1 != nil {
		return nil, xml_marshal_error("CreateBucketConfiguration", err1)
	}
	var i = types.CreateBucketConfiguration{
		Bucket:             o.Bucket,
		Location:           o.Location,
		LocationConstraint: o.LocationConstraint,
		Tags:               o.Tags.Tag,
	}
	return &i, nil
}

func export_CreateBucketConfiguration(e *xml.Encoder, i *types.CreateBucketConfiguration) error {
	var o = H_CreateBucketConfiguration{
		Bucket:             i.Bucket,
		Location:           i.Location,
		LocationConstraint: i.LocationConstraint,
		Tags: struct {
			Tag []types.Tag
		}{Tag: i.Tags},
	}
	var err1 = e.Encode(&o)
	if err1 != nil {
		return xml_marshal_error("CreateBucketConfiguration", err1)
	}
	return nil
}

// Marshaler of "TagSet []Tag" slot.
//
// It appears in types.Tagging.  The slot should be rendered as
// "<Tagging><TagSet><Tag>???</Tag>...</TagSet></Tagging>"

// The definition of "types.Tagging" in AWS-SDK is as follows and it
// will be rendered without <Tag> by the stdlib XML marshaler.
//
//   type Tagging struct {
//       TagSet []types.Tag
//   }

type H_Tagging struct {
	XMLName xml.Name `xml:"Tagging"`
	TagSet  H_TagSet
}

type H_TagSet struct {
	XMLName xml.Name `xml:"TagSet"`
	Tag     []types.Tag
}

func import_Tagging(d *xml.Decoder) (*types.Tagging, error) {
	var o H_Tagging
	var err1 = d.Decode(&o)
	if err1 != nil {
		return nil, xml_marshal_error("Tagging", err1)
	}
	var i = types.Tagging{TagSet: o.TagSet.Tag}
	return &i, nil
}

func export_Tagging(e *xml.Encoder, i *types.Tagging) error {
	var o = H_Tagging{TagSet: H_TagSet{Tag: i.TagSet}}
	var err1 = e.Encode(&o)
	if err1 != nil {
		return xml_marshal_error("Tagging", err1)
	}
	return nil
}

func import_TagSet(d *xml.Decoder) ([]types.Tag, error) {
	var o H_TagSet
	var err1 = d.Decode(&o)
	if err1 != nil {
		return nil, xml_marshal_error("TagSet", err1)
	}
	var i = o.Tag
	return i, nil
}

func export_TagSet(e *xml.Encoder, i []types.Tag) error {
	var o = H_TagSet{Tag: i}
	var err1 = e.Encode(&o)
	if err1 != nil {
		return xml_marshal_error("TagSet", err1)
	}
	return nil
}
