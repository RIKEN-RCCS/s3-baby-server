;; stub.scm (2025-10-16)

;; Ad-hoc server stub generator.  IT NEVER EVER GENERATES WORKING
;; CODE.  This generates dispatcher code for S3 requests.  It reads
;; "s3.json" in Smithy-2.0 and generates a skeleton for dispatcher
;; code.

;; This is for "guile --r7rs" and tested with GNU Guile-3.0.10.

;; Smithy syntax is described in: https://smithy.io/2.0/spec/idl.html
;; "+"-qualified element as in {Key+} matches one or more path
;; segments (never empty).  It is called greedy labels in
;; "14.1.2.4. Greedy labels".

;; ENTRY STRUCTURE OF "s3.json".  The outermost structure is:
;;
;; - {"metadata": {...}, "shapes": {...most of the contents...}}
;;
;; Entries of "shapes" part is:
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

;; The set of potentially required parameters (in query and header) is
;; small.  The set {"uploadId", "partNumber"} is for query.  The set
;; {"attributes", "delete", "list-type=2", "tagging", "uploads"} is
;; also query parameters but defined in a uri pattern.  The set
;; {"x-amz-copy-source", "x-amz-object-attributes"} is for header.
;; For example, query parameters are defined in
;; "/{Bucket}/{Key+}?tagging" (for "ObjectTagging"), and
;; "/{Bucket}?delete" (for "DeleteObjects").  "list-type=2" is for
;; ListObjectsV2.
;;
;; Some actions have its action name in query:
;; "x-id=AbortMultipartUpload" "x-id=CopyObject" "x-id=DeleteObject"
;; "x-id=GetObject" "x-id=ListBuckets" "x-id=ListParts"
;; "x-id=PutObject" "x-id=UploadPart" "x-id=UploadPartCopy"

;; Note on AWS SDK.  Input parameter "XXXXRequest" in API (and Smithy)
;; has a name "XXXXInput" in SDK.  "Request" is a wrapper of "Input"
;; used for invoking an actual remote call.

;; Golang http server: https://pkg.go.dev/net/http#ServeMux

(import
 (ice-9 exceptions)
 (ice-9 binary-ports)
 (ice-9 textual-ports)
 (ice-9 expect)
 (ice-9 popen)
 (ice-9 format)
 (ice-9 match)
 (ice-9 string-fun) ;; string-replace-substring
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

;; List of implemented actions of s3-baby-server.  The full list of S3
;; actions are listed in "shapes" / "com.amazonaws.s3#AmazonS3" /
;; "operations" in "s3.json".

(define list-of-action-names
  '(
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

(define tr? #t)

(define (assoc-option k alist)
  ;; Assoc but returns the cdr part of it or #f.  It #f as an alist.
  (if (eqv? alist #f)
      #f
      (cond ((assoc k alist)
	     => (lambda (pair) (cdr pair)))
	    (else #f))))

(define (assoc-with-default default k alist)
  ;; Assoc but returns the cdr part or default.
  (if (eqv? alist #f)
      default
      (cond ((assoc k alist) => (lambda (pair) (cdr pair)))
	    (else default))))

(define (lset-uniq eqvfn x)
   ;; This is delete-duplicates.
   (apply lset-adjoin eqvfn '() x))

(define (foldl f init list)
  ;; foldl : ('a * 'b -> 'b) -> 'b -> 'a list -> 'b
  ;; (f a4 (f a3 (f a2 (f a1 init))))
  (match list
    (() init)
    ((fst . rst) (foldl f (f fst init) rst))))

(define (foldr f init list)
  ;; foldr : ('a * 'b -> 'b) -> 'b -> 'a list -> 'b
  ;; (f a1 (f a2 (f a3 (f a4 init))))
  (match list
    (() init)
    ((fst . rst) (f fst (foldr f init rst)))))

(define (substitute-string s key val)
  ;; Replaces a key string with a val string in a string s.  A key is
  ;; regexp.  It replaces all occurrences of a key.
  (let* ((key1 (regexp-quote key)))
    (cond ((string-match key1 s)
	   => (lambda (m)
		(let* ((range (vector-ref m 1))
		       (prefix (substring s 0 (car range)))
		       (suffix (substring s (cdr range) (string-length s))))
		 (string-append
		  prefix
		  val
		  (substitute-string suffix key val)))))
	  (else s))))

(define (intervene-separator v separator)
  ;; Makes '(1 2 3) to '(1 sep 2 sep 3).
  (if (null? v)
      '()
      (let ((marker (cons 1 1)))
	(foldl (lambda (a b)
		 (if (eq? b marker)
		     (list a)
		     (append b (list separator)(list a))))
	       marker v))))

(define (append-strings v separator)
  ;; Appends strings with an intervening separator.
  (apply string-append (intervene-separator v separator)))

;;;
;;; LOADING S3.JSON
;;;

(display "Reading ./s3.json...\n")
(define s3-idl (with-input-from-file "./s3.json" json-read))
(define s3-api (cdr (assoc 'shapes s3-idl)))
(display "Reading ./s3.json... done\n")

;;;
;;; IDL SLOT ACCESSORS
;;;

;; See collected-actions.  Or try:
;;
;; (describe-action "AbortMultipartUpload")
;; (describe-action "DeleteObjects")
;; (describe-action "UploadPartCopy")

(define (find-action-structure action-name)
  (let ((key (string-append "com.amazonaws.s3#" action-name)))
    (assoc-option (string->symbol key) s3-api)))

(define (drop-prefix prefix name)
  ;; Drops the prefix part from the name string.  It assumes the name
  ;; begins with a prefix.
  (assert (string-prefix? prefix name))
  (substring name (string-length prefix)))

(define (rename-output-structure-name output)
  (string-replace-substring output "Output" "Response"))

(define (find-exchange-signature action-structure)
  ;; Returns a request/response name pair ("XXXXRequest"
  ;; "XXXXResponse").  It renames the result type "XXXXOutput" to
  ;; "XXXResponse".  It may return "Unit" for response.  Note the full
  ;; structure names look like: "com.amazonaws.s3#XXXXRequest" and
  ;; "com.amazonaws.s3#XXXXOutput".
  (let ((r1 (assoc-option 'target (assoc-option 'input action-structure)))
	(q1 (assoc-option 'target (assoc-option 'output action-structure)))
	(prefix "com.amazonaws.s3#"))
    (assert (and (string=? r1) (string=? q1)))
    (assert (not (string=? "smithy.api#Unit" r1)))
    (let ((r2 (drop-prefix prefix r1))
	  (q2 (cond ((string=? "smithy.api#Unit" q1)
		     "Unit")
		    (else
		     (drop-prefix prefix q1)))))
      (let ((q3 (string-replace-substring q2 "Output" "Response")))
	(list r2 q3)))))

(define (find-request-structure~ action-structure)
  (let* ((signature (find-exchange-signature action-structure))
	 (slot-name (string-append "com.amazonaws.s3#" (car signature))))
    (cdr (assoc (string->symbol slot-name) s3-api))))

(define (find-response-structure~ action-structure)
  (let* ((signature (find-exchange-signature action-structure))
	 (slot-name (string-append "com.amazonaws.s3#" (cadr signature))))
    (cdr (assoc (string->symbol slot-name) s3-api))))

;; Note the "traits" slot of an action-structure indicates the method
;; of a request.  It is under "smithy.api#http", and has the
;; properties of "method", "uri", "code".

(define (itemize-action-properties action-structure)
  ;; Extracts the method of an action.  It return a three-tuple of
  ;; (method uri-path-pattern code).  Method is a http method name,
  ;; and code is an http status code 200.
  (cond ((assoc-option '#{smithy.api#http}#
		       (assoc-option 'traits action-structure))
	 => (lambda (method)
	      (list (assoc-option 'method method)
		    (assoc-option 'uri method)
		    (assoc-option 'code method))))
	(else #f)))

;; Note the "traits" slot of a structure-slot indicates the location
;; of a request parameter.  It also indicates its required-ness.
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
;;   - indicates a slot is in payload body.
;; - "smithy.api#required": {}
;;   - indicates it is a required parameter.

(define (make-slot-properties member)
  ;; Admits an element of "members" list and returns a list of
  ;; four-tuples (required locus name slot) of a request/response
  ;; structure.  A name is in an xml tag, and a slot is a structure
  ;; slot name.
  (let* ((slot (symbol->string (car member)))
	 (traits (assoc-with-default '() 'traits (cdr member)))
	 (required (cond ((assoc '#{smithy.api#required}# traits) #t)
			 (else #f)))
	 (flatten (cond ((assoc '#{smithy.api#xmlFlattened}# traits) #t)
			(else #f)))
	 ;; (* IGNORE FLATTENED *)
	 (name (cond ((assoc '#{smithy.api#xmlName}# traits)
		      => (lambda (pair) (cdr pair)))
		     (else slot))))
    (cond ((assoc-option '#{smithy.api#httpLabel}# traits)
	   => (lambda (_)
		(list required 'path name slot)))
	  ((assoc-option '#{smithy.api#httpQuery}# traits)
	   => (lambda (v)
		(list required 'query v slot)))
	  ((assoc-option '#{smithy.api#httpHeader}# traits)
	   => (lambda (v)
		(list required 'header v slot)))
	  ((assoc-option '#{smithy.api#httpPrefixHeaders}# traits)
	   => (lambda (v)
		(list required 'header v slot)))
	  ((assoc-option '#{smithy.api#httpPayload}# traits)
	   => (lambda (_)
		;; (* DATA IS CONTENT PAYLOAD. *)
		(list required 'payload name slot)))
	  (else
	   ;; Empty traits means a response element.
	   (list required 'element name slot)))))

(define (itemize-slot-properties exchange-structure-name)
  ;; Extracts properties of a request/response structure of
  ;; "com.amazonaws.s3#XXXXRequest" and "com.amazonaws.s3#XXXXOutput".
  ;; It returns a list of four-tuples (required locus name slot), or
  ;; returns an empty list of "Unit".  A locus indicates where a
  ;; parameter is passed, and it is one of 'path, 'query, 'header,
  ;; 'payload, or 'element.  locus=payload means the value is a whole
  ;; payload.  A slot is a slot name of a request/response strucuture.
  (let* ((prefix "com.amazonaws.s3#")
	 (slot-name (string-append prefix exchange-structure-name))
	 (exchange-structure (assoc-option (string->symbol slot-name) s3-api))
	 (members (assoc-option 'members exchange-structure)))
    (if (eqv? #f members)
	'()
	(delete #f (map make-slot-properties members)))))

(define (adjust-input-structure-name request)
  (string-replace-substring request "Request" "Input"))

(define (adjust-output-structure-name response)
  (string-replace-substring response "Response" "Output"))

(define (describe-action action-name)
  ;; Returns a list of (action-name signature action-properties
  ;; request-properties response-properties).  A signature is a pair
  ;; of request/response names.  It renames the response name from
  ;; "XXXXOutput" to "XXXResponse".
  (when tr? (format #t ";; looking at action=~a~%" action-name))
  (let* ((action-structure (find-action-structure action-name))
	 (properties1 (itemize-action-properties action-structure))
	 (signature (find-exchange-signature action-structure))
	 (request-name (car signature))
	 (response-name (cadr signature))
	 (output-name (adjust-output-structure-name response-name))
	 (properties2 (itemize-slot-properties request-name))
	 (properties3 (itemize-slot-properties output-name)))
    (list action-name signature properties1 properties2 properties3)))

;; (itemize-slot-properties "ListPartsOutput")

(define collected-actions (map describe-action list-of-action-names))

;;;
;;; PARAMETER INQUERIES
;;;

;; See (collect-request-dispatches collected-actions).

(define (get-query-in-uri drop-x-id action-properties)
  ;; Finds an optional query key and returns it as a list: '(query) or
  ;; '().  It excludes "x-id"-key if drop-x-id is #t.  Query keys look
  ;; like: "/{Bucket}/{Key+}?uploads", "/{Bucket}/{Key+}?tagging",
  ;; "/{Bucket}?delete".
  (match-let (((method uri code) action-properties))
    ;;-(call-with-values (lambda () (apply values action-properties))
    ;;-  (lambda (method uri code)
    (let ((pat (regexp-quote "?")))
      (cond ((string-match pat uri)
	     => (lambda (m)
		  (let ((name (substring uri (+ (car (vector-ref m 1)) 1))))
		    (if (and drop-x-id (string-prefix? "x-id=" name))
			'()
			(list name)))))
	    (else '())))))

;; There are only a few occurring url patterns: {"/", "/{Bucket}",
;; "/{Bucket}/{Key+}"}.

(define (check-uri-prefix? prefix uri)
    (cond ((string-prefix? prefix uri)
	   (let ((n1 (string-length prefix))
		 (n2 (string-length uri)))
	     (assert (or (= n1 n2)
			 (and (> n2 n1)
			      (eqv? (string-ref uri n1) #\?))))
	     #t))
	  (else #f)))

(define (get-uri-method-path action-properties)
  ;; Returns a url method and pattern pair.  It replaces "/{Bucket}" to
  ;; "/{bucket}", "/{Bucket}/{Key+}" to "/{bucket}/{key...}", See the
  ;; "ServeMux" description for url patterns of Golang's httpd:
  ;; https://pkg.go.dev/net/http#ServeMux
  (match-let (((method uri code) action-properties))
    ;;-(call-with-values (lambda () (apply values action-properties))
    ;;-  (lambda (method uri code)
    (cond ((check-uri-prefix? "/{Bucket}/{Key+}" uri)
	   (list method "/{bucket}/{key...}"))
	  ((check-uri-prefix? "/{Bucket}" uri)
	   (list method "/{bucket}"))
	  ((check-uri-prefix? "/" uri)
	   (list method "/"))
	  (else
	   (format #t "BAD unknown url pattern found: ~s." uri)
	   #f))))

(define (make-request-dispatch action)
  ;; Makes a dispatch entry, and returns a list of (name method-path
  ;; queries headers signature).
  (match-let* (((name signature action-properties request-properties _) action)
	       (method-path (get-uri-method-path action-properties))
	       (query-in-uri (get-query-in-uri #t action-properties)))
    (let loop ((props request-properties)
	       (queries-acc query-in-uri)
	       (headers-acc '()))
      (if (null? props)
	  (list name method-path queries-acc headers-acc signature)
	  (match-let (((required locus name slot) (car props)))
	    (when tr? (format #t ";; required=~s locus=~s name=~s slot=~s~%"
			      required locus name slot))
	    (if (not required)
		(loop (cdr props) queries-acc headers-acc)
		(case locus
		  ((path)
		   (loop (cdr props) queries-acc headers-acc))
		  ((query)
		   (loop (cdr props)
			 (append queries-acc (list name))
			 headers-acc))
		  ((header)
		   (loop (cdr props)
			 queries-acc
			 (append headers-acc (list name))))
		  ((payload)
		   (loop (cdr props) queries-acc headers-acc))
		  ((element)
		   (loop (cdr props) queries-acc headers-acc))
		  (else
		   (format #t "BAD properties=~s~%" (car props))))))))))

(define (make-dispatch-entry action-name)
  (cond ((assoc action-name collected-actions)
	 => (lambda (action)
	      (make-request-dispatch action)))
	(else #f)))

;; (make-dispatch-entry "AbortMultipartUpload")
;; (make-dispatch-entry "DeleteObjects")
;; (make-dispatch-entry "UploadPartCopy")

(define (collect-request-dispatches collected-actions)
  (let loop ((actions collected-actions)
	     (acc '()))
    (if (null? actions)
	acc
	(let ((dispatch (make-request-dispatch (car actions))))
	  (loop (cdr actions) (append acc (list dispatch)))))))

(define collected-dispatches (collect-request-dispatches collected-actions))

;;;
;;; DISPATCHER PRINTER
;;;

;; This part prints a dispatch routine for requests collected by
;; collect-request-dispatches.  Collected requests are grouped by
;; mathod-path pairs.

;; MEMO about Golang net/http server.  Headers can be accessed in
;; Request.Header which is type Header (a map).  Queries can be
;; accessed in Request.URL.Query() which is type Values (a map).

;; RUN (display-dispatcher list-of-dispatches).

(define (merge-request-dispatches collected-actions)
  ;; Merges request dispatch entries by combining ones with the same
  ;; method-path pair.  It returns an alist with a method-path key and
  ;; a list of dispatches sharing the same key.
  (let loop ((entries collected-dispatches)
	     ;;(entries (collect-request-dispatches collected-actions))
	     (alist '()))
    (if (null? entries)
	alist
	(match-let* ((dispatch (car entries))
		     ((name method-path queries headers signature) dispatch))
	  (cond ((assoc method-path alist)
		 => (lambda (pair)
		      (loop (cdr entries)
			    (alist-cons method-path (cons dispatch (cdr pair))
					(alist-delete method-path alist)))))
		(else
		 (loop (cdr entries)
		       (alist-cons method-path (cons dispatch '())
				   alist))))))))

(define (dispatch-ordered? a b)
  (match-let (((_ _ queries-a headers-a _) a)
	      ((_ _ queries-b headers-b _) b))
    (> (+ (length queries-a) (length headers-a))
       (+ (length queries-b) (length headers-b)))))

(define method-ordering '("HEAD" "GET" "PUT" "POST" "DELETE"))

(define (method-path-ordered? alist-entry-a alist-entry-b)
    (match-let (((method-a path-a) (car alist-entry-a))
		((method-b path-b) (car alist-entry-b)))
      (let ((index-a (list-index (lambda (x) (string=? method-a x))
				 method-ordering))
	    (index-b (list-index (lambda (x) (string=? method-b x))
				 method-ordering))
	    (length-a (string-length path-a))
	    (length-b (string-length path-b)))
	(cond ((= index-a index-b)
	       (> length-a length-b))
	      (else
	       (< index-a index-b))))))

(define (sort-dispatches merged-dispatches)
  ;; Sorts the entries by their specific-ness, i.e., by the length of
  ;; queries and headers.  In addition, but not necessary, it also
  ;; sorts the alist by methods ordering in HEAD, GET, PUT, POST,
  ;; DELETE.
  (let loop1 ((alist merged-dispatches)
	      (dispatches-acc '()))
    (if (null? alist)
	(sort dispatches-acc method-path-ordered?)
	(match-let (((method-path . dispatches) (car alist)))
	  (let ((dispatches1 (sort dispatches dispatch-ordered?)))
	    (loop1 (cdr alist)
		   (alist-cons method-path dispatches1 dispatches-acc)))))))

(define merged-dispatches (merge-request-dispatches collected-actions))
(define list-of-dispatches (sort-dispatches merged-dispatches))

(define (make-registering-prologue method path)
  (format #f
	  "sx.HandleFunc(\"~a ~a\", ~a {"
	  method path "func(w http.ResponseWriter, r *http.Request)"))

(define (make-registering-epilogue)
  "else {http.NotFound(w, r) return}})")

(define (make-check-root-conditional)
  "if r.URL.Path != \"/\" {http.NotFound(w, r) return}")

(define (make-dispatch-choice q body)
  (format #f "else if ~a {~a}" q body))

(define (make-fetch-prologue)
  (values "var q = r.URL.Query()"
	  "var h = r.Header"))

(define (make-variable-name s)
  (string-map (lambda (c) (if (or (eqv? c #\-) (eqv? c #\=)) #\_ c))
	      (string-downcase s)))

(define (make-fetch-condition source s)
  (assert (or (string=? source "q") (string=? source "h")))
  (cond ((string-contains s "=")
	 => (lambda (i)
	      (let* ((key (substring s 0 i))
		     (var (make-variable-name s))
		     (val (substring s (+ i 1))))
		(format #f "var ~a = (~a.Get(~s) != ~s)" var source key val))))
	(else
	 (let* ((key s)
		(var (make-variable-name key)))
	   (format #f "var ~a = (~a.Get(~s) != \"\")" var source key)))))

(define (make-conditionals queries-headers)
  ;;(format #t "make-conditionals ~s~%" queries-headers)
  (if (null? queries-headers)
      "true"
      (let ((v (map make-variable-name queries-headers)))
	(string-append "(" (append-strings v " && ") ")"))))

(define (make-choice-clause dispatch)
  (match-let* (((name _ queries headers signature) dispatch)
	       (q (make-conditionals (append queries headers)))
	       (body (string-append "h_" name "(w, r)")))
    (make-dispatch-choice q body)))

(define (list-queries-headers dispatches)
  (let loop ((dispatches dispatches)
	     (queries-acc '())
	     (headers-acc '()))
    (if (null? dispatches)
	(values (sort (delete-duplicates queries-acc string=?) string<)
		(sort (delete-duplicates headers-acc string=?) string<))
	(match-let* ((dispatch (car dispatches))
		     ((name _ queries headers signature) dispatch))
	  (loop (cdr dispatches)
		(append queries-acc queries)
		(append headers-acc headers))))))

(define (display-dispatcher list-of-dispatches)
  ;; Prints pseudo code for "ServeMux" handler patterns.
  (format #t "{~%")
  (let loop1 ((dispatch-alist list-of-dispatches))
    (if (null? dispatch-alist)
	(values)
	(match-let ((((method path) . dispatches) (car dispatch-alist)))
	  (format #t "~a~%" (make-registering-prologue method path))
	  (when (string=? path "/")
	    (format #t "~a~%" (make-check-root-conditional)))
	  (let-values (((fetch-q fetch-h) (make-fetch-prologue)))
	    (format #t "~a~%" fetch-q)
	    (format #t "~a~%" fetch-h))
	  (let-values (((queries headers) (list-queries-headers dispatches)))
	    (let loop2 ((queries queries))
	      (if (null? queries)
		  (values)
		  (begin
		    (format #t "~a~%" (make-fetch-condition "q" (car queries)))
		    (loop2 (cdr queries)))))
	    (let loop3 ((headers headers))
	      (if (null? headers)
		  (values)
		  (begin
		    (format #t "~a~%" (make-fetch-condition "h" (car headers)))
		    (loop3 (cdr headers))))))
	  (format #t "if false {}~%")
	  (let loop4 ((dispatches dispatches))
	    (if (null? dispatches)
		(values)
		(begin
		  (format #t "~a~%" (make-choice-clause (car dispatches)))
		  (loop4 (cdr dispatches)))))
	  (format #t "~a~%" (make-registering-epilogue))
	  (loop1 (cdr dispatch-alist)))))
  (format #t "}~%"))

;; (display-dispatcher list-of-dispatches)

;;;
;;; HANDLER PRINTER
;;;

;; RUN (display-handler-call (assoc "ListParts" collected-dispatches))

(define (make-handler-prologue name)
  (list
   (format #f "func h_~a(~a, ~a, w http.ResponseWriter, r *http.Request) ~a {"
	   name "ctx context.Context" "bbs *service.S3Service2" "error")))

(define (make-handler-epilogue name)
  (list "}"))

(define (make-input-output-prologue request-name)
  (match-let* ((input (adjust-input-structure-name request-name)))
    (list "var qi = r.URL.Query()"
	  "var hi = r.Header"
	  "var ho = w.Header"
	  (format #f "var i = s3.~a{}" input))))

(define (locus-ordered? property-a property-b)
  (match-let (((required-a locus-a name-a slot-a) property-a)
	      ((required-b locus-b name-b slot-b) property-b))
    (cond ((eqv? locus-a 'payload)
	   #f)
	  ((eqv? locus-b 'payload)
	   #t)
	  (else
	   #t))))

(define (move-payload-assignment-to-tail request-properties)
  ;; Makes a payload assignment appear at the end, by sorting
  ;; request-properties.
  (sort request-properties locus-ordered?))

(define (make-input-assignment request-property)
  ;; Makes an assignment in a structure "s3.XXXXInput" of AWS-SDK.
  ;; Slot property is a list of (required locus name slot).  Note the
  ;; structure name of a request is "XXXXRequest" in the API and
  ;; Smithy.
  (match-let (((required locus name slot) request-property))
    ;; (when tr? (format #t ";; required=~s locus=~s name=~s slot=~s~%"
    ;; required locus name slot))
    (case locus
      ((path)
       ;; Ignore path parameters.
       '())
      ((query)
       (list (format #f "i.~a = qi.Get(~s)" slot name)))
      ((header)
       (list (format #f "i.~a = hi.Get(~s)" slot name)))
      ((payload)
       (list
	(format #f "{")
	(format #f "var x s3.~a" name)
	(format #f "var bs, err1 = io.ReadAll(r.Body)")
	(format #f "var err2 = xml.Unmarshal(bs, &x)")
	(format #f "if err2 != nil {return invalid_request()}")
	(format #f "i.~a = x" slot)
	(format #f "}")))
      ((element)
       (format #t "BAD properties=~s~%" request-property)
       (values))
      (else
       (format #t "BAD properties=~s~%" request-property)
       (values)))))

(define (make-input-assignments request-properties)
  (let* ((properties (move-payload-assignment-to-tail request-properties))
	 (s1 (map make-input-assignment properties)))
    (apply append s1)))

(define (make-handler-call name)
  (list
   (format #f "var o, err3 = bbs.~a(ctx, &i)" name)
   "if err3 != nil {bbs.logger.Error(\"\", \"error\", err3)"))

(define (make-output-extraction response-properties)
  ;; Makes extraction from structure "s3.XXXXOutput" of AWS SDK.  Each
  ;; slot property is a list of (required locus name slot).
  (let* (;;(input-name (adjust-input-structure-name response-name))
	 ;;(key (string-append "com.amazonaws.s3#" response-name))
	 ;;(response-structure (assoc-option (string->symbol key) s3-api))
	 ;;(props (itemize-slot-properties response-structure))
	 )
    (let loop ((props response-properties)
	       (acc '()))
      (if (null? props)
	  acc
	  (match-let (((required locus name slot) (car props)))
	    (format #t ";; required=~s locus=~s name=~s slot=~s~%"
		    required locus name slot)
	    (case locus
	      ((path)
	       (loop (cdr props) acc))
	      ((query)
	       (begin
		 (format #t "BAD query in response: ~s~%" name)
		 (values)))
	      ((header)
	       (let ((setter (format #f "ho.Add(~s, o.~a)" name slot)))
		 (loop (cdr props) (append acc (list setter)))))
	      ((payload)
	       (begin
		 (format #t "BAD payload in response: ~s~%" name)
		 (values)))
	      ((element)
	       (begin
		 (format #t "Skip element in response: ~s~%" name)
		 (loop (cdr props) acc)))
	      (else
	       (format #t "BAD properties=~s~%" (car props))
	       (values))))))))

;; (make-output-extraction "ListPartsOutput")

(define (make-output-payload-extraction)
  (list
   "ho.Set(\"Content-Type\", \"application/xml\")"
   "var co, err5 = xml.MarshalIndent(o, \" \", \"  \")"
   "if err5 != nil {bbs.logger.Error(\"\", \"error\", err5)}"
   "w.WriteHeader(status)"
   "var _, err6 = w.Write(co)"
   "if err6 != nil {bbs.Logger.Error(\"\", \"error\", err6)}"))

;; (make-input-assignments "PutObjectTaggingRequest")

(define (display-handler-call action)
  (match-let* (((name signature _ request-properties _) action)
	       ((request-name response-name) signature))
    (when tr? (format #t ";; make-repsonse-marshaler ~s~%" name))
    (let* ((s1 (make-handler-prologue name))
	   (s2 (make-input-output-prologue request-name))
	   (s3 (make-input-assignments request-properties))
	   (s5 (make-handler-call name))
	   ;;(s6 (make-output-extraction response-name))
	   ;;(s7 (make-output-payload-extraction))
	   (s8 (make-handler-epilogue name))
	   (ss (append s1 s2 s3 s5 s8)))
      (format #t "~a~%" (apply string-append (intervene-separator ss "\n"))))))

;; (display-handler-call (assoc "ListParts" collected-dispatches))

;;;
;;; RESPONSE MARSHALER PRINTER
;;;

(define (make-response-marshaler-preamble)
  (list
   (format #f "func p[T any](v T) *T {return &v}")
   (format #f "func s(k string) xml.StartElement {~a}"
	   "return xml.StartElement{Name: xml.Name{Local: k}")))

(define (check-output-in-payload response-properties)
  ;; Checks if response is in a payload.  "CopyObject" has
  ;; "CopyObjectResult", "GetObject" has "Body", and "UploadPartCopy"
  ;; has "CopyPartResult"
  (let ((check-output-in-payload1
	 (lambda (property)
	   (match-let (((required locus name slot) property))
	     (eqv? locus 'payload)))))
    (any check-output-in-payload1 response-properties)))

(define (make-slot-marshaler property)
  ;; Returns a list of marshaler lines of an response element.
  (match-let (((required locus name slot) property))
    (case locus
      ((path query)
       (format #t "BAD property in response: ~s~%" property)
       '())
      ((payload)
       (list
	(format #f "{var err2 = e.EncodeElement(r.~a, s(\"~a\"))" slot name)
	(format #f "if err2 != nil {return err2}}")))
      ((header)
       '())
      ((element)
       (list
	(format #f "{var err2 = e.EncodeElement(r.~a, s(\"~a\"))" slot name)
	(format #f "if err2 != nil {return err2}}")))
      (else
       (format #t "BAD property in response: ~s~%" property)
       '()))))

(define (make-repsonse-marshaler action)
  ;; Returns lines of response marshaler for "XXXXResponse" of an
  ;; argument action.
  (match-let*
      (((name (request-name response-name) _ _ response-properties) action)
       (output-name (adjust-output-structure-name response-name))
       (output-in-payload (check-output-in-payload response-properties))
       (encoders (delete '() (map make-slot-marshaler response-properties)))
       (nothing-in-payload (= (length encoders) 0)))
    (when tr? (format #t ";; make-repsonse-marshaler ~s~%" name))
    (assert (or (not output-in-payload) (= (length encoders) 1)))
    (append
     (list
      (format #f "type ~a s3.~a" response-name output-name)
      (format #f "func (r ~a) MarshalXML~a error {"
	      response-name "(e *xml.Encoder, start xml.StartElement)"))
     (if nothing-in-payload
	 '()
	 (append
	  (if output-in-payload
	      '()
	      (list
	       (format #f "var err1 = e.EncodeToken(start)")
	       (format #f "if err1 != nil {return err1}")))
	  (apply append encoders)
	  (if output-in-payload
	      '()
	      (list
	       (format #f "var err9 = e.EncodeToken(start.End())")
	       (format #f "if err9 != nil {return err9}")))))
     (list
      (format #f "return nil}")))))

(define (display-repsonse-marshaler1 action-name)
  (let* ((action (assoc action-name collected-actions))
	 (ss (make-repsonse-marshaler action))
	 (lines (apply string-append (intervene-separator ss "\n"))))
    (format #t "~a~%" lines)
    (values)))

(define (display-repsonse-marshaler)
  (let ((s1 (make-response-marshaler-preamble))
	(s2 (apply append (map make-repsonse-marshaler collected-actions))))
    (format #t "~a~%~a~%"
	    (apply string-append (intervene-separator s1 "\n"))
	    (apply string-append (intervene-separator s2 "\n")))))

;; (make-repsonse-marshaler (assoc "CopyObject" collected-actions))
;; (display-repsonse-marshaler)
