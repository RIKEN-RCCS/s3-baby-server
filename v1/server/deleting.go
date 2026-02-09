// deleting.go

// Copyright 2025-2026 RIKEN R-CCS
// SPDX-License-Identifier: BSD-2-Clause

// Deletion.  This is the common part of {DeleteObject,
// DeleteObjects}.

package server

import (
	"context"
	"io/fs"
	"os"
)

func (bbs *Bb_server) delete_object(ctx context.Context, object string, conditions copy_conditions) *Aws_s3_error {
	var location = "/" + object
	var _, rid = get_action_name(ctx)

	var entity string
	var etag string
	var stat fs.FileInfo

	{
		var err1, err2 error
		entity, stat, err1 = bbs.fetch_object_status(rid, object, false)
		if err1 != nil {
			// IGNORE-ERRORS.
		}
		if entity != "" {
			etag, _, err2 = bbs.fetch_object_etag(rid, object, entity)
			if err2 != nil {
				// IGNORE-ERRORS.
			}
		}
	}

	// SERIALIZE-ACCESSES.

	{
		var timeout = bbs.serialize_access(ctx, object, rid)
		if timeout != nil {
			return timeout
		}
		defer bbs.release_access(ctx, object, rid)
	}

	{
		var entity2, _, err3 = bbs.fetch_object_status(rid, object, true)
		if err3 != nil {
			// IGNORE-ERRORS.
		}
		if entity2 != entity {
			bbs.logger.Info("Race: Target object changed during operation",
				"rid", rid, "object", object)
			var errz = &Aws_s3_error{Code: InternalError,
				Message:  "Target object changed during operation.",
				Resource: location}
			return errz
		}

		var mtime = stat.ModTime()
		var size = stat.Size()
		var err7 = bbs.check_conditions(rid, object, etag,
			mtime, size, "delete", conditions)
		if err7 != nil {
			return err7
		}

		var err1 = bbs.store_object_metainfo(rid, object, nil)
		if err1 != nil {
			// IGNORE-ERRORS.
		}
		var path = bbs.make_path_of_object(object, "")
		var err2 = os.Remove(path)
		if err2 != nil {
			bbs.logger.Warn("os.Remove() on an object failed",
				"rid", rid, "path", path, "error", err2)
			return map_os_error(location, err2, nil)
		}
	}

	return nil
}
