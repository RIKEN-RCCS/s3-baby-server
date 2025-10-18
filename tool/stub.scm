;; stub.scm (2025-10-16)

;; Ad-hoc server stub generator.  IT DOES NOT GENERATE WORKING CODE.
;; This generates dispatcher code for S3 requests.  It reads "s3.json"
;; in Smithy-2.0 and generates a skeleton for dispatcher code.

;; This is for "guile --r7rs".  It is tested GNU Guile 3.0.10.

;; Smithy syntax is described in: https://smithy.io/2.0/spec/idl.html
;; "+"-qualified element as in {Key+} matches one or more path
;; segments (never empty).  It is called greedy labels in
;; "14.1.2.4. Greedy labels".

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
;; small.  The set {"uploadId", "partNumber"} for query.  The set
;; {"x-amz-copy-source", "x-amz-object-attributes"} for header.  Some
;; required query parameters are defined in a uri pattern:
;; {"attributes", "delete", "list-type=2", "tagging", "uploads"}.  For
;; example, "/{Bucket}/{Key+}?tagging" on "ObjectTagging", and
;; "/{Bucket}?delete" on "DeleteObjects".  "list-type=2" is for
;; ListObjectsV2.
;;
;; Some actions have its action name in query:
;; "x-id=AbortMultipartUpload" "x-id=CopyObject" "x-id=DeleteObject"
;; "x-id=GetObject" "x-id=ListBuckets" "x-id=ListParts"
;; "x-id=PutObject" "x-id=UploadPart" "x-id=UploadPartCopy"

;; Golang http server: https://pkg.go.dev/net/http#ServeMux

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

(define (assoc-option k alist)
  ;; Assoc but returns the cdr part or #f, also accepts #f as an
  ;; alist.
  (if (eqv? alist #f)
      #f
      (cond ((assoc k alist)
	     => (lambda (p) (cdr p)))
	    (else #f))))

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

;; See (collect-actions).  Or try:
;;
;; (describe-action "AbortMultipartUpload")
;; (describe-action "DeleteObjects")
;; (describe-action "UploadPartCopy")

(define (find-action-structure action-name)
  (let ((key (string-append "com.amazonaws.s3#" action-name)))
    (assoc-option (string->symbol key) s3-api)))

(define (find-request-and-response-name action-structure)
  ;; Returns a pair of input structure names of request and response,
  ;; like "XXXXRequest" and "XXXXOutput".  Note slot names look like:
  ;; "com.amazonaws.s3#XXXXRequest" and "com.amazonaws.s3#XXXXOutput".
  (let ((r1 (cdr (assoc 'target (cdr (assoc 'input action-structure)))))
	(q1 (cdr (assoc 'target (cdr (assoc 'output action-structure)))))
	(prefix "com.amazonaws.s3#"))
    (assert (not (string=? "smithy.api#Unit" r1)))
    (let ((r2 (cond (#t
		     (assert (string-prefix? prefix r1))
		     (substring r1 (string-length prefix)))))
	  (q2 (cond ((string=? "smithy.api#Unit" q1)
		     "Unit")
		    (else
		     (assert (string-prefix? prefix q1))
		     (substring q1 (string-length prefix))))))
      (list r2 q2))))

(define (find-request-structure action-structure)
  (let* ((names (find-request-and-response-name action-structure))
	 (slot (string-append "com.amazonaws.s3#" (car names))))
    (cdr (assoc (string->symbol slot) s3-api))))

(define (find-response-structure action-structure)
  (let* ((names (find-request-and-response-name action-structure))
	 (slot (string-append "com.amazonaws.s3#" (cadr names))))
    (cdr (assoc (string->symbol slot) s3-api))))

;; Note the "traits" slot of an action-structure indicates the method
;; of a request.  It is under "smithy.api#http", and has the
;; properties of "method", "uri", "code".

(define (get-action-properties action-structure)
  ;; Extracts the method of an action.  It return a list of method,
  ;; uri-path-pattern, and code.
  (cond ((assoc-option '#{smithy.api#http}#
		       (assoc-option 'traits action-structure))
	 => (lambda (method)
	      (list (assoc-option 'method method)
		    (assoc-option 'uri method)
		    (assoc-option 'code method))))
	(else #f)))

;; Note the "traits" slot of a structure-member indicates the location
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
;;   - indicates a slot is in body.
;; - "smithy.api#required": {}
;;   - indicates it is a required parameter.

(define (get-request-properties request-structure)
  ;; Extracts parameters of a request (e.g., of "PutObjectRequest").  The
  ;; locus where a parameter is passed is indicated in the traits
  ;; slot.  Locus is one of path, query, header, or body.
  (let ((members (cdr (assoc 'members request-structure))))
    (let loop ((members members)
	       (acc '()))
      (if (null? members)
	  acc
	  (let* ((e (car members))
		 (prop (get-member-properties e)))
	    (if (eqv? prop #f)
		(loop (cdr members) acc)
		(loop (cdr members) (append acc (cons prop '())))))))))

(define (get-member-properties m)
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
			      ;; (* body with #f is content payload. *)
			      (list required 'body n))))
		      (else #f)))))
	(else #f)))

(define (describe-action action-name)
  ;; Returns a list of (action-name request-response-names
  ;; action-properties request-properties).
  (format #t "looking at action=~a~%" action-name)
  (let* (;;(key (string-append "com.amazonaws.s3#" action-name))
	 ;;(assoc-option (string->symbol key) s3-api))
	 (action-structure (find-action-structure action-name))
	 (properties1 (get-action-properties action-structure))
	 (names (find-request-and-response-name action-structure))
	 (request (car names))
	 (response (cadr names))
	 (slot (string-append "com.amazonaws.s3#" request))
	 (request-structure (assoc-option (string->symbol slot) s3-api))
	 (properties2 (get-request-properties request-structure)))
    (list action-name names properties1 properties2)))

(define (collect-actions)
  (let loop ((names list-of-action-names)
	     (acc '()))
    (if (null? names)
	acc
	(let ((a (describe-action (car names))))
	  (loop (cdr names)
		(append acc (list a)))))))

(define collected-actions (collect-actions))

;;;
;;; PARAMETER INQUERIES
;;;

;; See (collect-request-dispatches collected-actions).

(define (get-query-in-uri ban-x-id action-properties)
  ;; Finds an optional query key and returns it as a list: '(query) or
  ;; '().  It excludes "x-id"-key if ban-x-id is #t.  Query keys look
  ;; like: "/{Bucket}/{Key+}?uploads", "/{Bucket}/{Key+}?tagging",
  ;; "/{Bucket}?delete".
  (match-let (((method uri code) action-properties))
    ;;-(call-with-values (lambda () (apply values action-properties))
    ;;-  (lambda (method uri code)
    (let ((pat (regexp-quote "?")))
      (cond ((string-match pat uri)
	     => (lambda (m)
		  (let ((name (substring uri (+ (car (vector-ref m 1)) 1))))
		    (if (and ban-x-id (string-prefix? "x-id=" name))
			'()
			(list name)))))
	    (else '())))))

(define (collect-all-required-parameters collected-actions)
  (let loop ((actions collected-actions)
	     (queries-acc '())
	     (headers-acc '()))
    (if (null? actions)
	(list (delete-duplicates queries-acc string=?)
	      (delete-duplicates headers-acc string=?))
	(match-let (((action-name request-response-names
				  action-properties request-properties)
		     (car actions)))
	  ;;-(call-with-values (lambda () (apply values (car actions)))
	  ;;-  (lambda (action-name request-response-names
	  ;;-		       action-properties request-properties)
	  (format #t "collect-all-required-parameters on ~s~%" action-name)
	  (let ((query-in-uri (get-query-in-uri #t action-properties)))
	    (let ((pair (collect-required-parameters request-properties)))
	      (loop (cdr actions)
		    (append queries-acc query-in-uri (car pair))
		    (append headers-acc (cadr pair)))))))))

(define (collect-required-parameters request-properties)
  (let loop ((props request-properties)
	     (queries-acc '())
	     (headers-acc '()))
    ;; (format #t "tuple=~s~%" props)
    (if (null? props)
	(list queries-acc headers-acc)
	(match-let (((required locus name) (car props)))
	  ;;- (call-with-values (lambda () (apply values (car props)))
	  ;;-   (lambda (required locus name)
	  (format #t "required=~s locus=~s name=~s~%" required locus name)
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
		((body)
		 (loop (cdr props) queries-acc headers-acc))
		(else
		 (format #t "BAD properties=~s~%" (car props)))))))))

(define parameter-pair (collect-all-required-parameters collected-actions))
(define required-queries (delete "x-id=" (car parameter-pair) string-prefix?))
(define required-headers (cadr parameter-pair))

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
  (match-let* (((name signature action-properties request-properties) action)
	       (method-path (get-uri-method-path action-properties))
	       (query-in-uri (get-query-in-uri #t action-properties)))
    (let loop ((props request-properties)
	       (queries-acc query-in-uri)
	       (headers-acc '()))
      (if (null? props)
	  (list name method-path queries-acc headers-acc signature)
	  (match-let (((required locus name) (car props)))
	    (format #t ";; required=~s locus=~s name=~s~%" required locus name)
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
		  ((body)
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
	(let ((pattern (make-request-dispatch (car actions))))
	  (loop (cdr actions) (append acc (list pattern)))))))

;; (define collected-dispatches (collect-request-dispatches collected-actions)

;;;
;;; STUB (DISPATCHER) PRINTER
;;;

;; See (diplay-dispatcher merged-dispatches).

;; About Golang net/http server.  Headers can be accessed in
;; Request.Header which is type Header (a map).  Queries can be
;; accessed in Request.URL.Query() which is type Values (a map).

(define (merge-request-dispatches collected-actions)
  ;; Merges request dispatch entries by combining ones with the same
  ;; method-path pair.  It returns an alist with a method-path key and
  ;; a list of dispatches sharing the same key.
  (let loop ((entries (collect-request-dispatches collected-actions))
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

(define (make-handler-prologue method path)
  (format #f
	  "sx.HandleFunc(\"~a ~a\", ~a {"
	  method path "func(w http.ResponseWriter, r *http.Request)"))

(define (make-handler-epilogue)
  "else {http.NotFound(w, r) return}})")

(define (make-check-root-conditional)
  "if r.URL.Path != \"/\" {http.NotFound(w, r) return}")

(define (make-handler-choice q body)
  (format #f "else if ~a {~a}" q body))

(define (make-fetch-prologue)
  (values "var q = r.URL.Query()"
	  "var h = r.Header"))

(define (make-variable-name s)
  (string-map (lambda (c) (if (or (eqv? c #\-) (eqv? c #\=)) #\_ c))
	      (string-downcase s)))

(define (make-fetch-conditional source s)
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

(define (make-fetch-query v)
  (make-fetch-conditional "q" v))

(define (make-fetch-header v)
  (make-fetch-conditional "h" v))

(define (make-conditionals queries-headers)
  ;;(format #t "make-conditionals ~s~%" queries-headers)
  (if (null? queries-headers)
      "true"
      (let ((v (map make-variable-name queries-headers)))
	(string-append "(" (append-strings v " && ") ")"))))

(define (make-choice-clause handler)
  (match-let* (((name _ queries headers signature) handler)
	       (q (make-conditionals (append queries headers)))
	       (body (string-append "h_" name "(w, r)")))
    (make-handler-choice q body)))

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

(define (diplay-dispatcher list-of-dispatches)
  ;; Prints pseudo code for "ServeMux" handler patterns.
  (format #t "{~%")
  (let loop1 ((dispatch-alist list-of-dispatches))
    (if (null? dispatch-alist)
	(values)
	(match-let ((((method path) . dispatches) (car dispatch-alist)))
	  (format #t "~a~%" (make-handler-prologue method path))
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
		    (format #t "~a~%" (make-fetch-query (car queries)))
		    (loop2 (cdr queries)))))
	    (let loop3 ((headers headers))
	      (if (null? headers)
		  (values)
		  (begin
		    (format #t "~a~%" (make-fetch-header (car headers)))
		    (loop3 (cdr headers))))))
	  (format #t "if false {}~%")
	  (let loop4 ((dispatches dispatches))
	    (if (null? dispatches)
		(values)
		(begin
		  (format #t "~a~%" (make-choice-clause (car dispatches)))
		  (loop4 (cdr dispatches)))))
	  (format #t "~a~%" (make-handler-epilogue))
	  (loop1 (cdr dispatch-alist)))))
  (format #t "}~%"))

;; (diplay-dispatcher list-of-dispatches)
