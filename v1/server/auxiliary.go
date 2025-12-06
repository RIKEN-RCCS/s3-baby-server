// auxiliary.go

// API-STUB.  This file is part of API-STUB.

package server

import (
	"encoding/xml"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"io"
)

func xml_import_error(ty string, e error) error {
	return fmt.Errorf("Marshal error for type %s with %w", ty, e)
}

// Marshaler of "Buckets []types.Bucket" slot.
//
// It is a slot in types.ListBucketsOutput.  The slot should be
// rendered as "<Buckets><Bucket>???</Bucket>...</Buckets>".

type H_Buckets struct {
	XMLName xml.Name `xml:"Buckets"`
	Bucket  []types.Bucket
}

func import_Buckets(r io.Reader) ([]types.Bucket, error) {
	var o H_Buckets
	var err1 = xml.NewDecoder(r).Decode(&o)
	if err1 != nil {
		return nil, xml_import_error("[]Bucket", err1)
	}
	var i = o.Bucket
	return i, nil
}

func export_Buckets(i []types.Bucket, w io.Writer) error {
	var o = H_Buckets{Bucket: i}
	var err1 = xml.NewEncoder(w).Encode(&o)
	if err1 != nil {
		return xml_import_error("[]Bucket", err1)
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
	Tag []types.Tag
}

func import_Tagging(r io.Reader) (*types.Tagging, error) {
	var o H_Tagging
	var err1 = xml.NewDecoder(r).Decode(&o)
	if err1 != nil {
		return nil, xml_import_error("Tagging", err1)
	}
	var i = types.Tagging{TagSet: o.TagSet.Tag}
	return &i, nil
}

func export_Tagging(i *types.Tagging, w io.Writer) error {
	var o = H_Tagging{TagSet: H_TagSet{i.TagSet}}
	var err1 = xml.NewEncoder(w).Encode(&o)
	if err1 != nil {
		return xml_import_error("Tagging", err1)
	}
	return nil
}
