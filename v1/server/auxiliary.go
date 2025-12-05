// auxiliary.go

// API-STUB.  This file is part of API-STUB.

package server

import (
	"encoding/xml"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"io"
)

// BB_ENUM_INTERN_ERROR is an error returned when interning
// an enumeration.
type Bb_xml_intern_error struct {
	Name string
}

func xml_intern_error(tag string, e error) error {
	return fmt.Errorf("Marshal error for type %s with %w", tag, e)
}

// The definition of "types.Tagging" in AWS-SDK:
//
//   type Tagging struct {
//       TagSet []types.Tag
//   }

type Tagging struct {
	TagSet TagSet
}

type TagSet struct {
	Tag []types.Tag
}

func intern_Tagging(r io.Reader) (*types.Tagging, error) {
	var o Tagging
	var err1 = xml.NewDecoder(r).Decode(&o)
	if err1 != nil {
		return nil, xml_intern_error("Tagging", err1)
	}
	var i = types.Tagging{TagSet: o.TagSet.Tag}
	return &i, nil
}

func extern_Tagging(i *types.Tagging, w io.Writer) error {
	var o = Tagging{TagSet{i.TagSet}}
	var err1 = xml.NewEncoder(w).Encode(&o)
	if err1 != nil {
		return xml_intern_error("Tagging", err1)
	}
	return nil
}
