// dispatcher.go (2025-11-17)
// API-STUB.  Dispatcher for net/http.ServeMux.  It
// switches handlers with regard to method-path patterns
// and required parameters in request API.
package server
import (
"net/http"
)
// REGISTER_DISPATCHER registers handers of BB-server to ServeMux.
func register_dispatcher(bbs *Bb_server, sx *http.ServeMux) error {
sx.HandleFunc("HEAD /{bucket}/{key...}", func(w http.ResponseWriter, r *http.Request) {
if true {h_HeadObject(bbs, w, r)} else {http.NotFound(w, r); return}})
sx.HandleFunc("HEAD /{bucket}", func(w http.ResponseWriter, r *http.Request) {
if true {h_HeadBucket(bbs, w, r)} else {http.NotFound(w, r); return}})
sx.HandleFunc("GET /{bucket}/{key...}", func(w http.ResponseWriter, r *http.Request) {
var q = r.URL.Query()
var attributes = q.Has("attributes")
var tagging = q.Has("tagging")
var uploadid = q.Has("uploadId")
var h = r.Header
var x_amz_object_attributes = (len(h.Values("x-amz-object-attributes")) == 0)
if attributes && x_amz_object_attributes {h_GetObjectAttributes(bbs, w, r)} else if uploadid {h_ListParts(bbs, w, r)} else if tagging {h_GetObjectTagging(bbs, w, r)} else if true {h_GetObject(bbs, w, r)} else {http.NotFound(w, r); return}})
sx.HandleFunc("GET /{bucket}", func(w http.ResponseWriter, r *http.Request) {
var q = r.URL.Query()
var list_type_2 = (q.Get("list-type") == "2")
var uploads = q.Has("uploads")
if list_type_2 {h_ListObjectsV2(bbs, w, r)} else if uploads {h_ListMultipartUploads(bbs, w, r)} else if true {h_ListObjects(bbs, w, r)} else {http.NotFound(w, r); return}})
sx.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
if true {h_ListBuckets(bbs, w, r)} else {http.NotFound(w, r); return}})
sx.HandleFunc("PUT /{bucket}/{key...}", func(w http.ResponseWriter, r *http.Request) {
var q = r.URL.Query()
var partnumber = q.Has("partNumber")
var tagging = q.Has("tagging")
var uploadid = q.Has("uploadId")
var h = r.Header
var x_amz_copy_source = (len(h.Values("x-amz-copy-source")) == 0)
if partnumber && uploadid && x_amz_copy_source {h_UploadPartCopy(bbs, w, r)} else if partnumber && uploadid {h_UploadPart(bbs, w, r)} else if tagging {h_PutObjectTagging(bbs, w, r)} else if x_amz_copy_source {h_CopyObject(bbs, w, r)} else if true {h_PutObject(bbs, w, r)} else {http.NotFound(w, r); return}})
sx.HandleFunc("PUT /{bucket}", func(w http.ResponseWriter, r *http.Request) {
if true {h_CreateBucket(bbs, w, r)} else {http.NotFound(w, r); return}})
sx.HandleFunc("POST /{bucket}/{key...}", func(w http.ResponseWriter, r *http.Request) {
var q = r.URL.Query()
var uploadid = q.Has("uploadId")
var uploads = q.Has("uploads")
if uploads {h_CreateMultipartUpload(bbs, w, r)} else if uploadid {h_CompleteMultipartUpload(bbs, w, r)} else {http.NotFound(w, r); return}})
sx.HandleFunc("POST /{bucket}", func(w http.ResponseWriter, r *http.Request) {
var q = r.URL.Query()
var delete = q.Has("delete")
if delete {h_DeleteObjects(bbs, w, r)} else {http.NotFound(w, r); return}})
sx.HandleFunc("DELETE /{bucket}/{key...}", func(w http.ResponseWriter, r *http.Request) {
var q = r.URL.Query()
var tagging = q.Has("tagging")
var uploadid = q.Has("uploadId")
if tagging {h_DeleteObjectTagging(bbs, w, r)} else if uploadid {h_AbortMultipartUpload(bbs, w, r)} else if true {h_DeleteObject(bbs, w, r)} else {http.NotFound(w, r); return}})
sx.HandleFunc("DELETE /{bucket}", func(w http.ResponseWriter, r *http.Request) {
if true {h_DeleteBucket(bbs, w, r)} else {http.NotFound(w, r); return}})
return nil}
