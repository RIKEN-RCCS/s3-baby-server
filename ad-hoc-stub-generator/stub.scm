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

(define (intervene-separator separator v)
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
  (apply string-append (intervene-separator separator v)))

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

(define (itemize-action-property action-structure)
  ;; Extracts the method of an action.  It return a three-tuple
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
  ;; four-tuples (slot name locus required) of a request/response
  ;; structure.  A slot is a name in a structure, and a name is in an
  ;; xml tag.
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
		(list slot name 'PATH required)))
	  ((assoc-option '#{smithy.api#httpQuery}# traits)
	   => (lambda (v)
		(list slot v 'QUERY required)))
	  ((assoc-option '#{smithy.api#httpHeader}# traits)
	   => (lambda (v)
		(list slot v 'HEADER required)))
	  ((assoc-option '#{smithy.api#httpPrefixHeaders}# traits)
	   => (lambda (v)
		(list slot v 'HEADER required)))
	  ((assoc-option '#{smithy.api#httpPayload}# traits)
	   => (lambda (_)
		;; (* DATA IS CONTENT PAYLOAD. *)
		(list slot name 'PAYLOAD required)))
	  (else
	   ;; Empty traits means a response element.
	   (list slot name 'ELEMENT required)))))

(define (itemize-slot-properties exchange-structure-name)
  ;; Extracts properties of a request/response structure of
  ;; "com.amazonaws.s3#XXXXRequest" and "com.amazonaws.s3#XXXXOutput".
  ;; It returns a list of four-tuples (slot name locus required), or
  ;; returns an empty list of "Unit".  A slot is a name in a
  ;; request/response strucuture.  A locus indicates where a parameter
  ;; is passed, and it is one of 'PATH, 'QUERY, 'HEADER, 'PAYLOAD, or
  ;; 'ELEMENT.  locus=PAYLOAD means the value is a whole payload.
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
  ;; Returns a list of (action-name signature action-property
  ;; request-properties response-properties).  A signature is a pair
  ;; of request/response names.  It renames the response name from
  ;; "XXXXOutput" to "XXXResponse".
  (when tr? (format #t ";; describe-action ~a~%" action-name))
  (let* ((action-structure (find-action-structure action-name))
	 (properties1 (itemize-action-property action-structure))
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

;; This part makes a list of dispatcher entries, which is later used
;; to print dispatcher code.
;;
;; See (collect-request-dispatches collected-actions)

(define (get-query-in-uri drop-x-id action-property)
  ;; Finds an optional query key and returns it as a list: (query) or
  ;; '().  It excludes "x-id"-key if drop-x-id is #t.  Query keys look
  ;; like: "/{Bucket}/{Key+}?uploads", "/{Bucket}/{Key+}?tagging",
  ;; "/{Bucket}?delete".
  (match-let (((method uri code) action-property))
    ;;-(call-with-values (lambda () (apply values action-property))
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

(define (get-uri-method-path action-property)
  ;; Returns a url method and pattern pair.  It replaces "/{Bucket}" to
  ;; "/{bucket}", "/{Bucket}/{Key+}" to "/{bucket}/{key...}", See the
  ;; "ServeMux" description for url patterns of Golang's httpd:
  ;; https://pkg.go.dev/net/http#ServeMux
  (match-let (((method uri code) action-property))
    ;;-(call-with-values (lambda () (apply values action-property))
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
  (match-let* (((name signature action-property request-properties _) action)
	       (method-path (get-uri-method-path action-property))
	       (query-in-uri (get-query-in-uri #t action-property)))
    (when tr? (format #t ";; make-request-dispatch ~s~%" name))
    (let loop ((props request-properties)
	       (queries-acc query-in-uri)
	       (headers-acc '()))
      (if (null? props)
	  (list name method-path queries-acc headers-acc signature)
	  (match-let (((slot name locus required) (car props)))
	    ;; (when tr? (format #t ";; slot=~s name=~s locus=~s required=~s~%"
	    ;; slot name locus required))
	    (if (not required)
		(loop (cdr props) queries-acc headers-acc)
		(case locus
		  ((PATH)
		   (loop (cdr props) queries-acc headers-acc))
		  ((QUERY)
		   (loop (cdr props)
			 (append queries-acc (list name))
			 headers-acc))
		  ((HEADER)
		   (loop (cdr props)
			 queries-acc
			 (append headers-acc (list name))))
		  ((PAYLOAD)
		   (loop (cdr props) queries-acc headers-acc))
		  ((ELEMENT)
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
;;
;; RUN (display-dispatcher list-of-dispatches)

;; MEMO about Golang net/http server.  Headers can be accessed in
;; Request.Header which is type Header (a map).  Queries can be
;; accessed in Request.URL.Query() which is type Values (a map).

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
	       (body (string-append "h_" name "(bbs, w, r)")))
    (format #f "if ~a {~a}" q body)))

(define (list-queries-headers dispatches)
  ;; Gathers queries and headers from dispatches.  The result is used
  ;; to access queries/headers in the dispacher code.
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

(define (generate-dispatcher-entry dispatch-entry)
  ;; Generate a dispatcher for one pattern.  It returns a list of line
  ;; strings.
  (match-let ((((method path) . dispatches) dispatch-entry))
    (let-values (((queries headers) (list-queries-headers dispatches)))
      (append
       ;; Handler registering code:
       (list
	(string-append
	 (format #f "sx.HandleFunc(\"~a ~a\"," method path)
	 " func(w http.ResponseWriter, r *http.Request) {"))
       ;; Check code for root-condition:
       (if (string=? path "/")
	   (list
	    "if r.URL.Path != \"/\" {http.NotFound(w, r); return}")
	   '())
       ;; Fetch code of queries:
       (if (null? queries)
	   '()
	   (append
	    (list "var q = r.URL.Query()")
	    (map (lambda (q) (make-fetch-condition "q" q))
		 queries)))
       ;; Fetch code of headers:
       (if (null? headers)
	   '()
	   (append
	    (list "var h = r.Header")
	    (map (lambda (h) (make-fetch-condition "h" h))
		 headers)))
       ;; Condition checks (in a single line):
       (list
	(apply string-append
	       (intervene-separator
		" "
		(append
		 (intervene-separator
		  "else"
		  (map make-choice-clause dispatches))
		 (list
		  "else {http.NotFound(w, r); return}})")))))))))

(define (generate-dispatcher list-of-dispatches)
  ;; Prints pseudo code for "ServeMux" handler patterns.
  (append
   (list "// dispather.go (2025-10-25)"
	 "package server"
	 "import ("
	 ;; "\"context\""
	 "\"net/http\""
	 ")"
	 ;; ***DUMMY***
	 "type BB_server struct {}"
	 (string-append
	  "func register_dispatcher"
	  "(bbs *BB_server, sx *http.ServeMux)"
	  " error {"))
   (apply
    append
    (map generate-dispatcher-entry list-of-dispatches))
   (list (format #f "return nil}"))))

(define (write-dispatcher port list-of-dispatches)
  (let ((ss (generate-dispatcher list-of-dispatches)))
    (format port "~a" (append-strings ss "\n"))))

(define (display-dispatcher)
  (write-dispatcher #t list-of-dispatches))

(define (dump-dispatcher file)
  (call-with-output-file file
    (lambda (port)
      (write-dispatcher port list-of-dispatches))))

;; (dump-dispatcher "dispacher.go")

;;;
;;; HANDLER PRINTER
;;;

;; This part prints handler functions, which are used in the
;; dispatcher.

(define (locus-ordered? property-a property-b)
  (match-let (((slot-a name-a locus-a required-a) property-a)
	      ((slot-b name-b locus-b required-b) property-b))
    (cond ((eqv? locus-a 'PAYLOAD)
	   #f)
	  ((eqv? locus-b 'PAYLOAD)
	   #t)
	  (else
	   #t))))

(define (cast-payload-property-rear request-properties)
  ;; Makes a payload assignment appear at the end for readability, by
  ;; sorting request-properties.
  (sort request-properties locus-ordered?))

(define (make-input-assignment request-property)
  ;; Makes an assignment in a structure "s3.XXXXInput" of AWS-SDK.
  ;; Slot property is a list of (slot name locus required).  Note the
  ;; structure name of a request is "XXXXRequest" in the API and
  ;; Smithy.
  (match-let (((slot name locus required) request-property))
    ;; (when tr? (format #t ";; required=~s locus=~s name=~s slot=~s~%"
    ;; required locus name slot))
    (case locus
      ((PATH)
       ;; Ignore path parameters.
       '())
      ((QUERY)
       (list (format #f "i.~a = qi.Get(~s)" slot name)))
      ((HEADER)
       (list (format #f "i.~a = hi.Get(~s)" slot name)))
      ((PAYLOAD)
       (list
	"{"
	(format #f "var x s3.~a" name)
	"var bs, err1 = io.ReadAll(r.Body)"
	"var err2 = xml.Unmarshal(bs, &x)"
	"if err2 != nil {return invalid_request()}"
	(format #f "i.~a = x" slot)
	"}"))
      ((ELEMENT)
       (format #t "BAD properties=~s~%" request-property)
       (values))
      (else
       (format #t "BAD properties=~s~%" request-property)
       (values)))))

(define (make-output-extraction response-property)
  ;; Makes extraction code from structure "s3.XXXXOutput" of AWS SDK.
  ;; Each property is a list of (slot name locus required).
  (match-let (((slot name locus required) response-property))
    ;; (when tr? (format #t ";; (slot=~s name=~s locus=~s required=~s)~%"
    ;; slot name locus required))
    (case locus
      ((PATH)
       '())
      ((QUERY)
       (begin
	 (format #t "BAD query in response: ~s~%" name)
	 (values)))
      ((HEADER)
       (list (format #f "ho.Add(~s, q.~a)" name slot)))
      ((PAYLOAD)
       (begin
	 (when tr? (format #t ";; Payload in response: ~s~%" name))
	 '()))
      ((ELEMENT)
       (begin
	 ;; (when tr? (format #t ";; Skip element in response: ~s~%" name))
	 '()))
      (else
       (format #t "BAD properties=~s~%" (car props))
       (values)))))

(define (make-output-payload-extraction code)
  (list
   "ho.Set(\"Content-Type\", \"application/xml\")"
   "var co, err5 = xml.MarshalIndent(q, \" \", \"  \")"
   "if err5 != nil {log.Fatal(err5); return err5}"
   (format #f "var status int = ~a" code)
   "w.WriteHeader(status)"
   "var _, err6 = w.Write(co)"
   "if err6 != nil {log.Fatal(err6); return err6}"))

(define (make-handler-definition action)
  (match-let*
      (((name signature action-property request-properties response-properties)
	action)
       ((request-name response-name) signature)
       (input-name (adjust-input-structure-name request-name))
       (output-name (adjust-output-structure-name response-name))
       (properties (cast-payload-property-rear request-properties))
       ((_ _ code) action-property))
    (when tr? (format #t ";; make-handler-definition ~s~%" name))
    (append
     (list
      ;; Start of function declaration:
      (string-append (format #f "func h_~a" name)
		     "(bbs *BB_server,"
		     " w http.ResponseWriter, r *http.Request) error {")
      "var qi = r.URL.Query()"
      "var hi = r.Header"
      "var ho = w.Header")
     ;; Input accessors:
     (list
      (format #f "var i = s3.~a{}" input-name))
     (apply append (map make-input-assignment properties))
     ;; Hander invocation:
     (list
      "var ctx = r.Context()"
      (format #f "var o, err3 = bbs.~a(ctx, &i)" name)
      "if err3 != nil {log.Fatal(err3); return err3}")
     ;; Output accessors:
     (list
      (format #f "var q = q_~a{s3.~a: o}" response-name output-name))
     (apply append (map make-output-extraction response-properties))
     (make-output-payload-extraction code)
     ;; Function end:
     (list "return nil}"))))

(define (display-handler-function action)
  (let ((ss (make-handler-definition action)))
    (format #t "~a~%" (apply string-append (intervene-separator "\n" ss)))))

(define (make-handler-file collected-actions)
  (append
   (list "// handlers.go (2025-10-25)"
	 "package server"
	 "import ("
	 ;; "\"context\""
	 "\"encoding/xml\""
	 "\"net/http\""
	 "\"log\""
	 "\"github.com/aws/aws-sdk-go-v2/service/s3\""
	 ")")
   (apply append
	  (map make-handler-definition collected-actions))))

(define (write-handlers port collected-actions)
  (let ((ss (make-handler-file collected-actions)))
    (format port "~a~%" (apply string-append (intervene-separator "\n" ss)))))

(define (display-handlers)
  (write-handlers #t collected-actions))

(define (dump-handlers file)
  (call-with-output-file file
    (lambda (port)
      (write-handlers port collected-actions))))

;; (display-handler-function (assoc "ListParts" collected-actions))
;; (dump-handlers "handlers.go")

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
	   (match-let (((slot name locus required) property))
	     (eqv? locus 'PAYLOAD)))))
    (any check-output-in-payload1 response-properties)))

(define (make-slot-marshaler property)
  ;; Returns a list of marshaler lines of an response element.
  (match-let (((slot name locus required) property))
    (case locus
      ((PATH QUERY)
       (format #t "BAD property in response: ~s~%" property)
       '())
      ((PAYLOAD)
       (list
	(format #f "{var err2 = e.EncodeElement(r.~a, s(\"~a\"))" slot name)
	(format #f "if err2 != nil {return err2}}")))
      ((HEADER)
       '())
      ((ELEMENT)
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
	 (lines (apply string-append (intervene-separator "\n" ss))))
    (format #t "~a~%" lines)
    (values)))

(define (display-repsonse-marshaler)
  (let ((s1 (make-response-marshaler-preamble))
	(s2 (apply append (map make-repsonse-marshaler collected-actions))))
    (format #t "~a~%~a~%"
	    (apply string-append (intervene-separator "\n" s1))
	    (apply string-append (intervene-separator "\n" s2)))))

;; (make-repsonse-marshaler (assoc "CopyObject" collected-actions))
;; (display-repsonse-marshaler)
