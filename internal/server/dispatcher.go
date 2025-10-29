// dispatcher.go (2025-10-01)
// Dispatcher for net/http.ServeMux.  It switches handlers
// with regard to method-path patterns and required
// parameters in request API.
package server

import (
	"net/http"
)

func register_dispatcher(bbs *BB_server, sx *http.ServeMux) error {
	sx.HandleFunc("HEAD /{bucket}/{key...}", func(w http.ResponseWriter, r *http.Request) {
		if true {
			h_HeadObject(bbs, w, r)
		} else {
			http.NotFound(w, r)
			return
		}
	})
	sx.HandleFunc("HEAD /{bucket}", func(w http.ResponseWriter, r *http.Request) {
		if true {
			h_HeadBucket(bbs, w, r)
		} else {
			http.NotFound(w, r)
			return
		}
	})
	sx.HandleFunc("GET /{bucket}/{key...}", func(w http.ResponseWriter, r *http.Request) {
		var q = r.URL.Query()
		var attributes = (q.Get("attributes") != "")
		var tagging = (q.Get("tagging") != "")
		var uploadid = (q.Get("uploadId") != "")
		var h = r.Header
		var x_amz_object_attributes = (h.Get("x-amz-object-attributes") != "")
		if attributes && x_amz_object_attributes {
			h_GetObjectAttributes(bbs, w, r)
		} else if uploadid {
			h_ListParts(bbs, w, r)
		} else if tagging {
			h_GetObjectTagging(bbs, w, r)
		} else if true {
			h_GetObject(bbs, w, r)
		} else {
			http.NotFound(w, r)
			return
		}
	})
	sx.HandleFunc("GET /{bucket}", func(w http.ResponseWriter, r *http.Request) {
		var q = r.URL.Query()
		var list_type_2 = (q.Get("list-type") != "2")
		var uploads = (q.Get("uploads") != "")
		if list_type_2 {
			h_ListObjectsV2(bbs, w, r)
		} else if uploads {
			h_ListMultipartUploads(bbs, w, r)
		} else if true {
			h_ListObjects(bbs, w, r)
		} else {
			http.NotFound(w, r)
			return
		}
	})
	sx.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		if true {
			h_ListBuckets(bbs, w, r)
		} else {
			http.NotFound(w, r)
			return
		}
	})
	sx.HandleFunc("PUT /{bucket}/{key...}", func(w http.ResponseWriter, r *http.Request) {
		var q = r.URL.Query()
		var partnumber = (q.Get("partNumber") != "")
		var tagging = (q.Get("tagging") != "")
		var uploadid = (q.Get("uploadId") != "")
		var h = r.Header
		var x_amz_copy_source = (h.Get("x-amz-copy-source") != "")
		if partnumber && uploadid && x_amz_copy_source {
			h_UploadPartCopy(bbs, w, r)
		} else if partnumber && uploadid {
			h_UploadPart(bbs, w, r)
		} else if tagging {
			h_PutObjectTagging(bbs, w, r)
		} else if x_amz_copy_source {
			h_CopyObject(bbs, w, r)
		} else if true {
			h_PutObject(bbs, w, r)
		} else {
			http.NotFound(w, r)
			return
		}
	})
	sx.HandleFunc("PUT /{bucket}", func(w http.ResponseWriter, r *http.Request) {
		if true {
			h_CreateBucket(bbs, w, r)
		} else {
			http.NotFound(w, r)
			return
		}
	})
	sx.HandleFunc("POST /{bucket}/{key...}", func(w http.ResponseWriter, r *http.Request) {
		var q = r.URL.Query()
		var uploadid = (q.Get("uploadId") != "")
		var uploads = (q.Get("uploads") != "")
		if uploads {
			h_CreateMultipartUpload(bbs, w, r)
		} else if uploadid {
			h_CompleteMultipartUpload(bbs, w, r)
		} else {
			http.NotFound(w, r)
			return
		}
	})
	sx.HandleFunc("POST /{bucket}", func(w http.ResponseWriter, r *http.Request) {
		var q = r.URL.Query()
		var delete = (q.Get("delete") != "")
		if delete {
			h_DeleteObjects(bbs, w, r)
		} else {
			http.NotFound(w, r)
			return
		}
	})
	sx.HandleFunc("DELETE /{bucket}/{key...}", func(w http.ResponseWriter, r *http.Request) {
		var q = r.URL.Query()
		var tagging = (q.Get("tagging") != "")
		var uploadid = (q.Get("uploadId") != "")
		if tagging {
			h_DeleteObjectTagging(bbs, w, r)
		} else if uploadid {
			h_AbortMultipartUpload(bbs, w, r)
		} else if true {
			h_DeleteObject(bbs, w, r)
		} else {
			http.NotFound(w, r)
			return
		}
	})
	sx.HandleFunc("DELETE /{bucket}", func(w http.ResponseWriter, r *http.Request) {
		if true {
			h_DeleteBucket(bbs, w, r)
		} else {
			http.NotFound(w, r)
			return
		}
	})
	return nil
}
