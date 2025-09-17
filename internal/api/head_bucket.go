// Copyright 2025-2025 RIKEN R-CCS\.
// SPDX-License-Identifier: BSD-2-Clause

package api

import (
	"net/http"
	"s3-baby-server/internal/service"
)

func HeadBucketHandler(s3 *service.S3Service) S3HandlerFunc {
	return func(w http.ResponseWriter, options *HTTPS3Options) *service.S3Error {
		options.Logger.Info("HeadBucket")
		if _, err := s3.HeadBucket(options); err != nil {
			return err
		}
		w.Header().Set("X-Amz-Access-Point-Alias", "false")
		ResponseStatusHeader(w, *options, http.StatusOK)
		return nil
	}
}
