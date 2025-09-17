// Copyright 2025-2025 RIKEN R-CCS\.
// SPDX-License-Identifier: BSD-2-Clause

package api

import (
	"net/http"
	"s3-baby-server/internal/service"
)

func ListPartsHandler(s3 *service.S3Service) S3HandlerFunc {
	return func(w http.ResponseWriter, options *HTTPS3Options) *service.S3Error {
		options.Logger.Info("ListParts")
		result, err := s3.ListParts(options)
		if err != nil {
			return err
		}
		ResponseWriteHeader(result, w, *options, http.StatusOK)
		return nil
	}
}
