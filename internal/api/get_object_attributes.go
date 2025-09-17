// Copyright 2025-2025 RIKEN R-CCS\.
// SPDX-License-Identifier: BSD-2-Clause

package api

import (
	"net/http"
	"s3-baby-server/internal/service"
	"time"
)

func GetObjectAttributesHandler(s3 *service.S3Service) S3HandlerFunc {
	return func(w http.ResponseWriter, options *HTTPS3Options) *service.S3Error {
		options.Logger.Info("GetObjectAttributes")
		result, err := s3.GetObjectAttributes(options)
		if err != nil {
			return err
		}
		options.Logger.Debug("", "handle", result.GetObjectAttributesContents)
		w.Header().Set("Last-Modified", result.LastModified.Format(time.RFC3339))
		ResponseWriteHeader(result.GetObjectAttributesContents, w, *options, http.StatusOK)
		return nil
	}
}
