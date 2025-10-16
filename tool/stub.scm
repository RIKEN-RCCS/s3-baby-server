;; stub.scm (2025-10-16)

;; Ad-hoc server stub generator.  This generates dispatcher code for
;; S3 requests.  It reads "s3.json" in Smithy-2.0 and generates a
;; skeleton for dispatcher code.  I don't know about Smithy's code
;; generators for Golang.

;; This is for "guile --r7rs".  It is tested GNU Guile 3.0.10.

;; Smithy syntax is described in: https://smithy.io/2.0/spec/idl.html
;; "+"-qualified element as in {Key+} matches more than one path
;; segment.  It is called greedy labels.

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

;; The sets of possibly required parameters (of query and header) are
;; small.  For query: {"uploadId", "partNumber"} and for header:
;; {"x-amz-copy-source", "x-amz-object-attributes"}.  Note Actions on
;; ObjectTagging have uri pattern: "/{Bucket}/{Key+}?tagging", and
;; "tagging" does not appear in the query set.  Similarily,
;; "DeleteObjects" has uri pattern: "/{Bucket}?delete" but

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
		  "AbortMultipartUpload"
		  "CompleteMultipartUpload"
		  "CopyObject"
		  "CreateBucket"
		  "CreateMultipartUpload"
		  "DeleteBucket"
		  "DeleteObject"
		  "DeleteObjects"
		  "DeleteObjectTagging"
		  "GetObject"
		  "GetObjectAttributes"
		  "GetObjectTagging"
		  "HeadBucket"
		  "HeadObject"
		  "ListBuckets"
		  "ListMultipartUploads"
		  "ListObjects"
		  "ListObjectsV2"
		  "ListParts"
		  "PutObject"
		  "PutObjectTagging"
		  "UploadPart"
		  "UploadPartCopy"))

(define (assoc-option k alist)
  ;; Assoc but returns the cdr part or #f, also accepts #f as an
  ;; alist.
  (if (eqv? alist #f)
      #f
      (cond ((assoc k alist)
	     => (lambda (p) (cdr p)))
	    (else #f))))

(define (find-action-structure action-name)
  (let ((key (string-append "com.amazonaws.s3#" action-name)))
    (assoc-option (string->symbol key) s3api)))

(define (find-request-and-response-name action-structure)
  ;; Returns a pair of input structure names of request and response,
  ;; like "XXXXRequest" and "XXXXOutput".  Note slot names look like:
  ;; "com.amazonaws.s3#XXXXRequest" and "com.amazonaws.s3#XXXXOutput".
  (let ((r1 (cdr (assoc 'target (cdr (assoc 'input action-structure)))))
	(q1 (cdr (assoc 'target (cdr (assoc 'output action-structure)))))
	(prefix "com.amazonaws.s3#"))
    (assert (not (string=? "smithy.api#Unit" r1)))
    (let ((r2 (cond (#t
		     (assert (string=? (substring r1 0 (string-length prefix))
				       prefix))
		     (substring r1 (string-length prefix)))))
	  (q2 (cond ((string=? "smithy.api#Unit" q1)
		     "Unit")
		    (else
		     (assert (string=? (substring q1 0 (string-length prefix))
				       prefix))
		     (substring q1 (string-length prefix))))))
      (list r2 q2))))

(define (find-request-structure action-structure)
  (let* ((names (find-request-and-response-name action-structure))
	 (slot (string-append "com.amazonaws.s3#" (car names))))
    (cdr (assoc (string->symbol slot) s3api))))

(define (find-response-structure action-structure)
  (let* ((names (find-request-and-response-name action-structure))
	 (slot (string-append "com.amazonaws.s3#" (cadr names))))
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
  (let ((members (cdr (assoc 'members request-structure))))
    (let loop ((members members)
	       (acc '()))
      (if (null? members)
	  acc
	  (let* ((e (car members))
		 (prop (member-properties e)))
	    (if (eqv? prop #f)
		(loop (cdr members) acc)
		(loop (cdr members) (append acc (cons prop '())))))))))

;; Note the "traits" slot of a structure-member indicates the location
;; of a request parameter.  It also indicates required-ness.
;;
;; - "smithy.api#httpLabel": {}
;;   - indicates a slot is in URL path.
;; - "smithy.api#httpQuery": "uploadId"
;;   - indicates a slot is in URL query.
;; - "smithy.api#httpHeader": "x-amz-request-payer"
;;   - indicates a slot is in header.
;; - "smithy.api#httpPrefixHeaders": "x-amz-meta-"
;;   - indicates a slot is in header.
;; - "smithy.api#httpPayload": {}
;;   - indicates a slot is in body, like in DeleteObjects.
;; - "smithy.api#required": {}
;;   - indicates it is a required parameter.

(define (member-properties m)
  ;; Admits an element of a "member" slot, and returns a list of
  ;; (required path/query/header/body name) of a request parameters.
  (cond ((assoc-option 'traits (cdr m))
	 => (lambda (r)
	      (let ((required
		     (cond ((assoc '#{smithy.api#required}# r)
			    #t)
			   (else #f))))
		(cond ((assoc-option '#{smithy.api#httpLabel}# r)
		       => (lambda (_)
			    (list required 'path (symbol->string (car m)))))
		      ((assoc-option '#{smithy.api#httpQuery}# r)
		       => (lambda (v)
			    (list required 'query v)))
		      ((assoc-option '#{smithy.api#httpHeader}# r)
		       => (lambda (v)
			    (list required 'header v)))
		      ((assoc-option '#{smithy.api#httpPrefixHeaders}# r)
		       => (lambda (v)
			    (list required 'header v)))
		      ((assoc-option '#{smithy.api#httpPayload}# r)
		       => (lambda (_)
			    (let ((n (assoc-option '#{smithy.api#xmlName}# r)))
			      (list required 'body n))))
		      (else #f)))))
	(else #f)))

(define (describe-action action-name)
  ;; Returns a list of (action-name request-response-names
  ;; action-properties parameter-properties).
  (format #t "looking at action=~a~%" action-name)
  (let* (;;(key (string-append "com.amazonaws.s3#" action-name))
	 ;;(assoc-option (string->symbol key) s3api))
	 (action-structure (find-action-structure action-name))
	 (properties1 (action-properties action-structure))
	 (names (find-request-and-response-name action-structure))
	 (request (car names))
	 (response (cadr names))
	 (slot (string-append "com.amazonaws.s3#" request))
	 (request-structure (assoc-option (string->symbol slot) s3api))
	 (properties2 (request-member-properties request-structure)))
    (list action-name names properties1 properties2)))

;; (describe-action "AbortMultipartUpload")
;; (describe-action "DeleteObjects")
;; (describe-action "UploadPartCopy")

(define (collect-actions)
  (let loop ((names actions)
	     (acc '()))
    (if (null? names)
	acc
	(let ((a (describe-action (car names))))
	  (loop (cdr names)
		(append acc (list a)))))))

(define collected-actions (collect-actions))

(define (collect-requied-parameters actions)
  (let loop ((tuples actions)
	     (query-acc '())
	     (header-acc '()))
    (if (null? tuples)
	(list query-acc header-acc)
	(call-with-values (lambda () (apply values (car tuples)))
	  (lambda (action-name request-response-names
			       action-properties parameter-properties)
	    (format #t "collect-requied-parameters on ~s~%" action-name)
	    (let ((pair (required-parameters parameter-properties)))
	      (loop (cdr tuples)
		    (append query-acc (car pair))
		    (append header-acc (cadr pair)))))))))

(define (required-parameters parameter-properties)
  (let loop ((tuples parameter-properties)
	     (query-acc '())
	     (header-acc '()))
    (format #t "tuple=~s~%" tuples)
    (if (null? tuples)
	(list query-acc header-acc)
	(call-with-values (lambda () (apply values (car tuples)))
	  (lambda (required locus name)
	    (format #t "required=~s locus=~s name=~s)~%"
		    required locus name)
	    (if (not required)
		(loop (cdr tuples) query-acc header-acc)
		(case locus
		  ((path)
		   (loop (cdr tuples) query-acc header-acc))
		  ((query)
		   (loop (cdr tuples)
			 (append query-acc (list name))
			 header-acc))
		  ((header)
		   (loop (cdr tuples)
			 query-acc
			 (append header-acc (list name))))
		  ((body)
		   (loop (cdr tuples) query-acc header-acc))
		  (else
		   (format #t "BAD properties=~s~%" (car tuples))))))))))

(collect-requied-parameters collected-actions)
