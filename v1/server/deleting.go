// deleting.go

// Copyright 2025-2026 RIKEN R-CCS
// SPDX-License-Identifier: BSD-2-Clause

// Deletion.  This is the common part of {DeleteObject,
// DeleteObjects}.

package server

import (
	//"bytes"
	"context"
	//"crypto/md5"
	//"encoding/base64"
	//"encoding/binary"
	//"encoding/hex"
	//"encoding/json"
	//"encoding/xml"
	//"errors"
	//"fmt"
	//"github.com/aws/aws-sdk-go-v2/service/s3/types"
	//"hash"
	//"io"
	//"io/fs"
	//"log"
	//"time"
	//"net/url"
	"os"
	//"path"
	//"path/filepath"
	//"strings"
)

func (bbs *Bb_server) delete_object(ctx context.Context, object string, conditionals copy_conditionals) *Aws_s3_error {
	var location = "/" + object
	var _, rid = get_action_name(ctx)
	// var rid uint64 = get_request_id(ctx)

	// SERIALIZE-ACCESSES.

	{
		var timeout = bbs.serialize_access(ctx, object, rid)
		if timeout != nil {
			return timeout
		}
		defer bbs.release_access(ctx, object, rid)
	}

	{
		var _, _, err3 = bbs.check_object_exists(object)
		if err3 != nil {
			return err3
		}

		var err5 = bbs.check_request_conditionals(object, "delete",
			conditionals)
		if err5 != nil {
			return err5
		}

		var err1 = bbs.store_metainfo(object, nil)
		if err1 != nil {
			// IGNORE-ERRORS.
		}
		var path = bbs.make_path_of_object(object, "")
		var err2 = os.Remove(path)
		if err2 != nil {
			bbs.logger.Warn("os.Remove() failed on an object",
				"path", path, "error", err2)
			return map_os_error(location, err2, nil)
		}
	}

	return nil
}
