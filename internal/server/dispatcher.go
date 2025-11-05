// dispatcher.go (2025-10-01)
// Dispatcher for net/http.ServeMux.  It switches handlers
// with regard to method-path patterns and required
// parameters in request API.
package server

import (
	"fmt"
	"net/http"
	//"s3-baby-server/internal/service"
)

func register_dispatcher(bbs *Bb_server, sx *http.ServeMux) error {
	sx.HandleFunc("HEAD /{bucket}/{key...}", func(w http.ResponseWriter, r *http.Request) {
		if true {
			fmt.Printf("h_HeadObject!\n")
			//h_HeadObject(bbs, w, r)
			bbs.HeadObjectHandler(w, r)
		} else {
			http.NotFound(w, r)
			return
		}
	})
	sx.HandleFunc("HEAD /{bucket}", func(w http.ResponseWriter, r *http.Request) {
		if true {
			fmt.Printf("h_HeadBucket!\n")
			//h_HeadBucket(bbs, w, r)
			bbs.HeadBucketHandler(w, r)
		} else {
			http.NotFound(w, r)
			return
		}
	})
	sx.HandleFunc("GET /{bucket}/{key...}", func(w http.ResponseWriter, r *http.Request) {
		var q = r.URL.Query()
		var attributes = q.Has("attributes")
		var tagging = q.Has("tagging")
		var uploadid = q.Has("uploadId")
		var h = r.Header
		var x_amz_object_attributes = (h.Get("x-amz-object-attributes") != "")
		if attributes && x_amz_object_attributes {
			fmt.Printf("h_GetObjectAttributes!\n")
			//h_GetObjectAttributes(bbs, w, r)
			bbs.GetObjectAttributesHandler(w, r)
		} else if uploadid {
			fmt.Printf("h_ListPart!\n")
			//h_ListParts(bbs, w, r)
			bbs.ListPartsHandler(w, r)
		} else if tagging {
			fmt.Printf("h_GetObjectTagging!\n")
			//h_GetObjectTagging(bbs, w, r)
			bbs.GetObjectTaggingHandler(w, r)
		} else if true {
			fmt.Printf("h_GetObject!\n")
			//h_GetObject(bbs, w, r)
			bbs.GetObjectHandler(w, r)
		} else {
			http.NotFound(w, r)
			return
		}
	})
	sx.HandleFunc("GET /{bucket}", func(w http.ResponseWriter, r *http.Request) {
		var q = r.URL.Query()
		var list_type_2 = (q.Get("list-type") == "2")
		var uploads = q.Has("uploads")
		if list_type_2 {
			fmt.Printf("h_ListObjectsV2!\n")
			//h_ListObjectsV2(bbs, w, r)
			bbs.ListObjectsV2Handler(w, r)
		} else if uploads {
			fmt.Printf("h_ListMultipartUploads!\n")
			//h_ListMultipartUploads(bbs, w, r)
			bbs.ListMultipartUploadsHandler(w, r)
		} else if true {
			fmt.Printf("h_ListObjects!\n")
			//h_ListObjects(bbs, w, r)
			bbs.ListObjectsHandler(w, r)
		} else {
			http.NotFound(w, r)
			return
		}
	})
	sx.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		if true {
			fmt.Printf("h_ListBuckets!\n")
			//h_ListBuckets(bbs, w, r)
			bbs.ListBucketsHandler(w, r)
		} else {
			http.NotFound(w, r)
			return
		}
	})
	sx.HandleFunc("PUT /{bucket}/{key...}", func(w http.ResponseWriter, r *http.Request) {
		var q = r.URL.Query()
		var partnumber = q.Has("partNumber")
		var tagging = q.Has("tagging")
		var uploadid = q.Has("uploadId")
		var h = r.Header
		var x_amz_copy_source = (h.Get("x-amz-copy-source") != "")
		if partnumber && uploadid && x_amz_copy_source {
			fmt.Printf("h_UploadPartCopy!\n")
			//h_UploadPartCopy(bbs, w, r)
			bbs.UploadPartCopyHandler(w, r)
		} else if partnumber && uploadid {
			fmt.Printf("h_UploadPart!\n")
			//h_UploadPart(bbs, w, r)
			bbs.UploadPartHandler(w, r)
		} else if tagging {
			fmt.Printf("h_PutObjectTagging!\n")
			//h_PutObjectTagging(bbs, w, r)
			bbs.PutObjectTaggingHandler(w, r)
		} else if x_amz_copy_source {
			fmt.Printf("h_CopyObject!\n")
			//h_CopyObject(bbs, w, r)
			bbs.CopyObjectHandler(w, r)
		} else if true {
			fmt.Printf("h_PutObject!\n")
			//h_PutObject(bbs, w, r)
			bbs.PutObjectHandler(w, r)
		} else {
			http.NotFound(w, r)
			return
		}
	})
	sx.HandleFunc("PUT /{bucket}", func(w http.ResponseWriter, r *http.Request) {
		if true {
			fmt.Printf("h_CreateBucket!\n")
			h_CreateBucket(bbs, w, r)
			//bbs.CreateBucketHandler(w, r)
		} else {
			http.NotFound(w, r)
			return
		}
	})
	sx.HandleFunc("POST /{bucket}/{key...}", func(w http.ResponseWriter, r *http.Request) {
		var q = r.URL.Query()
		var uploadid = q.Has("uploadId")
		var uploads = q.Has("uploads")
		if uploads {
			fmt.Printf("h_CreateMultipartUpload!\n")
			//h_CreateMultipartUpload(bbs, w, r)
			bbs.CreateMultipartUploadHandler(w, r)
		} else if uploadid {
			fmt.Printf("h_CompleteMultipartUpload!\n")
			//h_CompleteMultipartUpload(bbs, w, r)
			bbs.CompleteMultipartUploadHandler(w, r)
		} else {
			http.NotFound(w, r)
			return
		}
	})
	sx.HandleFunc("POST /{bucket}", func(w http.ResponseWriter, r *http.Request) {
		var q = r.URL.Query()
		var delete = q.Has("delete")
		if delete {
			fmt.Printf("h_DeleteObjects!\n")
			//h_DeleteObjects(bbs, w, r)
			bbs.DeleteObjectsHandler(w, r)
		} else {
			http.NotFound(w, r)
			return
		}
	})
	sx.HandleFunc("DELETE /{bucket}/{key...}", func(w http.ResponseWriter, r *http.Request) {
		var q = r.URL.Query()
		var tagging = q.Has("tagging")
		var uploadid = q.Has("uploadId")
		if tagging {
			fmt.Printf("h_DeleteObjectTagging!\n")
			//h_DeleteObjectTagging(bbs, w, r)
			bbs.DeleteObjectTaggingHandler(w, r)
		} else if uploadid {
			fmt.Printf("h_AbortMultipartUpload!\n")
			//h_AbortMultipartUpload(bbs, w, r)
			bbs.AbortMultipartUploadHandler(w, r)
		} else if true {
			fmt.Printf("h_DeleteObject!\n")
			//h_DeleteObject(bbs, w, r)
			bbs.DeleteObjectHandler(w, r)
		} else {
			http.NotFound(w, r)
			return
		}
	})
	sx.HandleFunc("DELETE /{bucket}", func(w http.ResponseWriter, r *http.Request) {
		if true {
			fmt.Printf("h_DeleteBucket!\n")
			//h_DeleteBucket(bbs, w, r)
			bbs.DeleteBucketHandler(w, r)
		} else {
			http.NotFound(w, r)
			return
		}
	})
	return nil
}
