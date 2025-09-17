// Copyright 2025-2025 RIKEN R-CCS\.
// SPDX-License-Identifier: BSD-2-Clause

package api

import (
	"net/http"
	"s3-baby-server/internal/service"
)

func DeleteObjectHandler(s3 *service.S3Service) S3HandlerFunc {
	return func(w http.ResponseWriter, options *HTTPS3Options) *service.S3Error {
		if options.GetKey() == "" { // keyが空ならDeleteBucketの処理を呼び出し
			DeleteBucketHandler(s3)
			return nil
		}
		options.Logger.Info("DeleteObject")
		err := s3.DeleteObject(options)
		if err != nil {
			return err
		}
		ResponseStatusHeader(w, *options, http.StatusNoContent)
		return nil
	}
}
