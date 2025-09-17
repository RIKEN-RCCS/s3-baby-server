// Copyright 2025-2025 RIKEN R-CCS\.
// SPDX-License-Identifier: BSD-2-Clause

package model

type DeleteObjectsState struct {
	ReqBody    DeleteRequest
	DeleteList []string
	Deleted    []ObjectKey
	Error      []ObjectKey
}

func (d DeleteObjectsState) MakeDeleteObjectsOptionsResult() *DeleteObjectsResult {
	result := DeleteObjectsResult{
		Deleted: d.Deleted,
	}
	if !d.ReqBody.Quiet {
		result.Error = d.Error // 削除に失敗したファイルリスト
	}
	return &result
}
