;; stub.scm (2025-10-16)

;; Server stub generator.  This generates ad-hoc dispatcher code of S3
;; requests.  It reads "s3.json" in Smithy-2.0 and generates a
;; skeleton for dispatcher code.  Smithy's code generators (for
;; Golang, etc.) are likely yet not ready for general use.

;; This is for "guile --r7rs".  It is tested GNU Guile 3.0.10.

;; ENTRY STRUCTURE OF "s3.json".  Outer most structure:
;;
;; - {"metadata": {...}, "shapes": {...most of the contents...}}
;;
;; Entries of "shapes" part:
;;
;; - "com.amazonaws.s3#Bucket": {"type": "structure", ...}
;; - "com.amazonaws.s3#UploadPart": {"type": "operation", ...}
;;
;; Entries of "com.amazonaws.s3#UploadPart":
;;
;; - "type": "operation",
;; - "input": {"target": "com.amazonaws.s3#UploadPartRequest"},
;; - "output": {"target": "com.amazonaws.s3#UploadPartOutput"},
;; - "traits": {
;;   - "smithy.api#http": {
;;     - "method": "PUT",
;;     - "uri": "/{Bucket}/{Key+}?x-id=UploadPart",
;;     - ...

(import
 (ice-9 exceptions)
 (ice-9 binary-ports)
 (ice-9 textual-ports)
 (ice-9 expect)
 (ice-9 popen)
 (ice-9 format)
 (ice-9 match)
 ;;(scheme base)
 (only (scheme base) define-record-type textual-port? write-string)
 (only (scheme base) vector-map vector-for-each vector->list)
 ;;(srfi srfi-133) ;; r7rs-vector-library (NO srfi-133 in Guile)
 (only (rnrs base) infinite? assert)
 (srfi srfi-1) ;; list
 (srfi srfi-11) ;; multiple-values
 ;;(srfi srfi-28) ;; format
 (srfi srfi-60) ;; integers as bits
 )

(setlocale LC_ALL "C.utf-8")

(define (assume . bs) '())
(define (%read-error? x)
  (read-error? x))
(define (valid-number? string)
  (number? (string->number string)))

(load "../test/minima/srfi-180-body.scm")

(define s3idl (with-input-from-file "./s3.json" json-read))

(define s3api (cdr (assoc 'shapes s3idl)))

;; List of implemented actions.  The full list of S3 actions are
;; listed in "shapes" ."com.amazonaws.s3#AmazonS3" ."operations" in
;; "s3.json".

(define actions '(
		  AbortMultipartUpload
		  CompleteMultipartUpload
		  CopyObject
		  CreateBucket
		  CreateMultipartUpload
		  DeleteBucket
		  DeleteObject
		  DeleteObjects
		  DeleteObjectTagging
		  GetObject
		  GetObjectAttributes
		  GetObjectTagging
		  HeadBucket
		  HeadObject
		  ListBuckets
		  ListMultipartUploads
		  ListObjects
		  ListObjectsV2
		  ListParts
		  PutObject
		  PutObjectTagging
		  UploadPart
		  UploadPartCopy))

(define (assoc-option k alist)
  ;; Assoc but returns the cdr part or #f, also accepts #f as an
  ;; alist.
  (if (eqv? alist #f)
      #f
      (cond ((assoc k alist)
	     => (lambda (p) (cdr p)))
	    (else #f))))

(define (find-request-and-response action-structure)
  ;; Returns a pair of input structure names of request and response,
  ;; like "XXXXRequest" and "XXXXOutput".  Note slot names look like:
  ;; "com.amazonaws.s3#XXXXRequest" and "com.amazonaws.s3#XXXXOutput".
  (let ((r (cdr (assoc 'target (cdr (assoc 'input action-structure)))))
	(q (cdr (assoc 'target (cdr (assoc 'output action-structure)))))
	(prefix "com.amazonaws.s3#"))
    (assert (string=? (substring r 0 (string-length prefix)) prefix))
    (assert (string=? (substring q 0 (string-length prefix)) prefix))
    (list (substring r (string-length prefix))
	  (substring q (string-length prefix)))))

(define (find-request-structure action-structure)
  (let* ((names (find-request-and-response action-structure))
	 (slot (string-append "com.amazonaws.s3#" (car names))))
    (cdr (assoc (string->symbol slot) s3api))))

;; Note the "traits" slot of an action-structure indicates the method
;; of a request.  It is under "smithy.api#http", and has the
;; properties of "method", "uri", "code".

(define (action-properties action-structure)
  ;; Extracts the method of an action.  It return a list of method,
  ;; uri-path-pattern, and code.
  (cond ((assoc-option '#{smithy.api#http}#
		       (assoc-option 'traits action-structure))
	 => (lambda (method)
	      (list (assoc-option 'method method)
		    (assoc-option 'uri method)
		    (assoc-option 'code method))))
	(else #f)))

(define (request-member-properties request-structure)
  (let ((m (cdr (assoc 'members request-structure))))
    (let loop ((m m)
	       (slots '()))
      (if (null? m)
	  slots
	  (let* ((e (car m))
		 (prop (member-properties e)))
	    (loop (cdr m) (append slots (cons prop '()))))))))

;; Note the "traits" slot of a structure-member indicates the location
;; of a request parameter.  It also indicates required-ness.
;;
;; - Example: "smithy.api#required": {}
;;   - indicates it is a required parameter.
;; - Example: "smithy.api#httpLabel": {}
;;   - indicates the slot is in URL path.
;; - Example: "smithy.api#httpQuery": "uploadId"
;;   - indicates the slot is in URL query.
;; - Example: "smithy.api#httpHeader": "x-amz-request-payer"
;;   - indicates the slot is in header.

(define (member-properties m)
  ;; Admits an element of a "member" slot, and returns a list of
  ;; (required path/query/header name) of a request parameters.
  (cond ((assoc 'traits (cdr m))
	 => (lambda (r)
	      (let ((required
		     (cond ((assoc '#{smithy.api#required}# (cdr r))
			    #t)
			   (else #f))))
		(cond ((assoc '#{smithy.api#httpLabel}# (cdr r))
		       => (lambda (p)
			    (list required 'path (symbol->string (car m)))))
		      ((assoc '#{smithy.api#httpQuery}# (cdr r))
		       => (lambda (p)
			    (list required 'query (cdr p))))
		      ((assoc '#{smithy.api#httpHeader}# (cdr r))
		       => (lambda (p)
			    (list required 'header (cdr p))))
		      (else #f)))))
	(else #f)))

(define (describe-action action-name)
  (let* ((key (string-append "com.amazonaws.s3#" action-name))
	 (action-structure (assoc-option (string->symbol key) s3api))
	 (properties1 (action-properties action-structure))
	 (names (find-request-and-response action-structure))
	 (request (car names))
	 (response (cadr names))
	 (slot (string-append "com.amazonaws.s3#" request))
	 (request-structure (assoc-option (string->symbol slot) s3api))
	 (properties2 (request-member-properties request-structure)))
    (list action-name names properties1 properties2)))

(define AbortMultipartUpload (cdr (assoc '#{com.amazonaws.s3#AbortMultipartUpload}# s3api)))

(describe-action "AbortMultipartUpload")
'(action-properties AbortMultipartUpload)
'(structure-members AbortMultipartUploadRequest)
