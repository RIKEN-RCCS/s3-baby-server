// Copyright 2025-2025 RIKEN R-CCS\.
// SPDX-License-Identifier: BSD-2-Clause

package api

import (
	"net/http"
	"s3-baby-server/internal/service"
)

func HeadObjectHandler(s3 *service.S3Service) S3HandlerFunc {
	return func(w http.ResponseWriter, options *HTTPS3Options) *service.S3Error {
		options.Logger.Info("HeadObject")
		result, err := s3.HeadObject(options)
		if err != nil {
			return err
		}
		ResponseGetHeader(result, w, *options, http.StatusOK, false) // GetObjectと同様の処理のため流用
		return nil
	}
}
