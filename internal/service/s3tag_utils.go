// Copyright 2025-2025 RIKEN R-CCS\.
// SPDX-License-Identifier: BSD-2-Clause

package service

import (
	"encoding/json"
	"os"
	"s3-baby-server/internal/model"
	"s3-baby-server/pkg/utils"
	"strconv"
	"strings"
	"unicode/utf16"
	"unicode/utf8"
)

type Tag struct {
	FileSystem       *FileSystem
	DirectiveCopy    string
	DirectiveReplace string
}

const (
	maxTagKey   = 128
	maxTagValue = 256
	maxTagCount = 10
)

func (t *Tag) validateTags(tags []model.Tag) (int, int) {
	var okCnt, ngCnt int
	for _, t := range tags {
		if !utf8.ValidString(t.Key) || !utf8.ValidString(t.Value) {
			ngCnt++
			continue
		}
		okCnt++
	}
	return okCnt, ngCnt
}

func (t *Tag) taggingDirective(option S3Options, srcPath, dstPath string) *S3Error {
	v := option.GetOption("x-amz-tagging-directive")
	srcMetaAPath := t.FileSystem.getMetaFileName(t.FileSystem.getFullPath(srcPath))
	dstMetaAPath := t.FileSystem.getMetaFileName(t.FileSystem.getFullPath(dstPath))
	tagging, flg := t.copyTagging(option)
	if !flg {
		return InvalidTag()
	}
	switch v {
	case t.DirectiveCopy:
		if !t.FileSystem.copyFile(srcMetaAPath, dstMetaAPath, 0, 0) {
			return InternalError()
		}
	case t.DirectiveReplace:
		file, err := os.Create(dstMetaAPath)
		if err != nil {
			return InternalError()
		}
		defer utils.CloseFile(file, t.FileSystem.Logger)
		t.FileSystem.Logger.Debug("tagging directive", "value", tagging)
		encoder := json.NewEncoder(file)
		encoder.SetIndent("", "  ")
		if err = encoder.Encode(tagging); err != nil {
			return InternalError()
		}
	}
	return nil
}

func utf16Len(s string) int {
	runes := []rune(s)
	encoded := utf16.Encode(runes)
	return len(encoded)
}

func validateTagKeyValue(key, value string) *S3Error {
	if key == "" {
		return InvalidTag()
	}
	if utf16Len(key) > maxTagKey {
		return InvalidTag()
	}
	if utf16Len(value) > maxTagValue {
		return InvalidTag()
	}
	return nil
}

func (t *Tag) validateTagSet(tags []model.Tag) *S3Error {
	if len(tags) == 0 {
		t.FileSystem.Logger.Error("tagset must be one of: key, value")
		return InvalidTag()
	}
	if len(tags) > maxTagCount {
		t.FileSystem.Logger.Error("tags cannot be more than 10", "", tags)
		return InvalidTag()
	}
	for _, tg := range tags {
		if err := validateTagKeyValue(tg.Key, tg.Value); err != nil {
			t.FileSystem.Logger.Error("invalid tag key/value", "key", tg.Key, "value", tg.Value)
			return err
		}
	}
	return nil
}

func (t *Tag) putTagging(option S3Options, dstPath string) *S3Error {
	v := option.GetOption("x-amz-tagging")
	if v == "" {
		return nil
	}
	t.FileSystem.Logger.Debug("tagging", "value", v)
	tag := t.getURLTagSet(v)
	if err := t.validateTagSet(tag.TagSet.Tags); err != nil {
		return err
	}
	cnt := 0
	for _, t := range tag.TagSet.Tags {
		if t.Key == "TagSet" {
			value := t.Value
			for i := 0; i <= len(value)-5; i++ {
				if value[i:i+5] == "{Key=" {
					cnt++
				}
			}
			break
		}
	}
	if cnt > maxTagCount {
		t.FileSystem.Logger.Error("Tags cannot be more than 10", "", tag.TagSet.Tags)
		return BadRequest()
	}
	dstAPath := t.FileSystem.getFullPath(dstPath)
	dstMetaPath := t.FileSystem.getMetaFileName(dstAPath)
	t.FileSystem.Logger.Debug(dstMetaPath)
	file, err := os.Create(dstMetaPath)
	if err != nil {
		return InternalError()
	}
	defer utils.CloseFile(file, t.FileSystem.Logger)
	e := json.NewEncoder(file)
	e.SetIndent("", "  ")
	if err = e.Encode(tag); err != nil {
		return InternalError()
	}
	return nil
}

func (t *Tag) putOnlyTagging(v model.Tagging, dstPath string) *S3Error {
	if err := t.validateTagSet(v.TagSet.Tags); err != nil {
		return err
	}
	dstAPath := t.FileSystem.getFullPath(dstPath)
	dstMetaPath := t.FileSystem.getMetaFileName(dstAPath)
	file, err := os.Create(dstMetaPath)
	if err != nil {
		return InternalError()
	}
	defer utils.CloseFile(file, t.FileSystem.Logger)
	e := json.NewEncoder(file)
	e.SetIndent("", "  ")
	if err = e.Encode(v); err != nil {
		return InternalError()
	}
	return nil
}

func (t *Tag) getTagCount(path string) (string, string) {
	if ok, ng := t.tagCounter(path); ok != 0 {
		return strconv.Itoa(ok), strconv.Itoa(ng)
	}
	return "", ""
}

func (t *Tag) tagCounter(path string) (int, int) {
	file, err := os.Open(t.FileSystem.getFullPath(t.FileSystem.getMetaFileName(path)))
	if err != nil {
		return 0, 0
	}
	defer utils.CloseFile(file, t.FileSystem.Logger)
	d := json.NewDecoder(file)
	var data model.Tagging
	if err = d.Decode(&data); err != nil {
		return 0, 0
	}
	return t.validateTags(data.TagSet.Tags)
}

func (t *Tag) readTag(path string) *model.Tagging {
	if !t.FileSystem.isFileExists(t.FileSystem.getMetaFileName(path)) {
		return &model.Tagging{TagSet: model.TagSet{Tags: []model.Tag{}}}
	}
	file, err := os.Open(t.FileSystem.getFullPath(t.FileSystem.getMetaFileName(path)))
	if err != nil {
		return nil
	}
	defer utils.CloseFile(file, t.FileSystem.Logger)
	var data model.Tagging
	if err = json.NewDecoder(file).Decode(&data); err != nil {
		return nil
	}
	t.FileSystem.Logger.Debug("", "変換後", data)
	return &data
}

func (t *Tag) copyTagging(option S3Options) (*model.Tagging, bool) {
	value := option.GetOption("x-amz-tagging")
	if value == "" {
		return nil, true
	}
	tagging := t.getURLTagSet(value)
	if len(tagging.TagSet.Tags) == 0 {
		t.FileSystem.Logger.Error("TagSet must be one of: Key, Value")
		return nil, false
	}
	return &tagging, true
}

func (t *Tag) getURLTagSet(value string) model.Tagging {
	tags := t.extractURLTags(value)
	t.FileSystem.Logger.Debug("Get TagSet", "value", tags)
	if len(tags) == 0 {
		return model.Tagging{}
	}
	return model.Tagging{TagSet: model.TagSet{Tags: tags}}
}

func (t *Tag) extractURLTags(value string) []model.Tag {
	var tags []model.Tag
	pairs := strings.Split(value, "&")
	for _, pair := range pairs {
		kv := strings.SplitN(pair, "=", 2)
		if len(kv) != 2 {
			continue // フォーマットが不正ならスキップ
		}
		key := strings.TrimSpace(kv[0])
		val := strings.TrimSpace(kv[1])
		tags = append(tags, model.Tag{
			Key:   key,
			Value: val,
		})
	}
	t.FileSystem.Logger.Debug("Extract TagSet", "value", tags)
	return tags
}
