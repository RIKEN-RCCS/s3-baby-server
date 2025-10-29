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

;; "s3.json" STRUCTURE.  The outermost structure is:
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

;; ASSUPTION: It assumes enumeration types are string types.  Null
;; value for enumeration types is "".

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

(define apply-append concatenate)
(define append-strings string-join)
;; substitute-string is done by string-replace-substring

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

#|
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
|#

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

#|
(define (append-strings v separator)
  ;; Appends strings with an intervening separator.
  (apply string-append (intervene-separator separator v)))
|#

(define (drop-prefix prefix name)
  ;; Drops the prefix part from the name string.  It assumes the name
  ;; begins with a prefix.
  (assert (string-prefix? prefix name))
  (substring name (string-length prefix)))

(define (string-split-n strings chars)
  ;; Repeats string-split by characters on strings.
  (if (null? chars)
      strings
      (string-split-n
       (apply append (map (lambda (s) (string-split s (car chars))) strings))
       (cdr chars))))

(define (capitalize-string s)
  ;; Capitalizes the string, when the string is all lowercase or
  ;; uppercase.  That is, it keeps camelcase.  It accepts zero-length
  ;; strings.
  (if (and (not (string-null? s))
	   (or (string=? (string-upcase s) s)
	       (string=? (string-downcase s) s)))
      (string-append (string (char-upcase (string-ref s 0)))
		     (string-downcase (substring s 1)))
      s))

(define (camelcase-string s)
  ;; Makes a string in camelcase, as "BabyIron", "baby_iron", and
  ;; "BABY_IRON" to "BabyIron".
  (let ((tokens (string-split-n (list s) '(#\_ #\- #\:))))
    (apply string-append (map capitalize-string tokens))))

;;;
;;; LOADING "s3.json"
;;;

(display "Reading ./s3.json...\n")
(define s3-idl (with-input-from-file "./s3.json" json-read))
(define s3-api (cdr (assoc 'shapes s3-idl)))
(display "Reading ./s3.json... done\n")

;;;
;;; LISTING TYPES
;;;

;; This part makes a list of type-defintions, LIST-OF-TYPES.  A
;; type-defintion is (type-name type-kind . slot-property...), where a
;; slot-property describes members of composite-types.  Each
;; slot-property is a five-tuple: (slot name type locus required).

;; Types in S3: {"blob", "boolean", "enum"*, "integer", "list"*,
;; "long", "map"*, ("operation"), ("service"), "string", "structure"*,
;; "timestamp", "union"*, "Unit"}.
;;
;; Types stared (*) are composite.  There are many defined types: 335
;; structures, 3 unions, 70 emulations, and one map.  Types
;; parenthesised above (operation and service) are meta-information.

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

;; Note the "Unit"-type is not listed in the list-of-types.

;; Note string types are stored as a pointer in AWS-SDK.

(define list-of-primitive-types
  '("blob" "boolean" "integer" "long" "operation" "service" "string"
    "timestamp"))

(define (primitive-type? type-kind)
  (cond ((member type-kind list-of-primitive-types)
	 #t)
	(else
	 ;; (format #t "non primitive-type ~s~%" type-kind)
	 #f)))

(define (assoc-type-of-slot default property)
  ;; Returns a type of the slot which is specified by "target".  It
  ;; accepts #f for property alist.
  (cond ((assoc-option 'target property)
	 => (lambda (target)
	      (cond ((string=? "smithy.api#Unit" target)
		     "Unit")
		    (else
		     (drop-prefix "com.amazonaws.s3#" target)))))
	(else default)))

(define (make-type-slot-property~ member)
  (match-let (((slot-symbol . property) member))
    (let* ((slot (symbol->string slot-symbol))
	   (type (assoc-type-of-slot slot property))
	   (traits (assoc-with-default '() 'traits property))
	   (name
	    (cond ((assoc '#{smithy.api#xmlName}# traits)
		   => (lambda (pair) (cdr pair)))
		  (else slot))))
      (list slot name type))))

(define (make-slot-property member)
  ;; Admits an element of "members" and returns a five-tuple (slot
  ;; name type locus required), describing a structure slot.  A slot
  ;; is a name in a structure (specified by key part), and a name is
  ;; an xml-tag (specified by "smithy.api#xmlName").  A type is a type
  ;; name (specified by "target").  A locus and a required are only
  ;; meaningful in request/response structures.  A locus indicates
  ;; where a parameter is passed, and it is one of {PATH, QUERY,
  ;; HEADER, HEADER-PREFIX, PAYLOAD, ELEMENT}.  locus=PAYLOAD means
  ;; the value is a whole payload.  A name is an enumerator for
  ;; enumeration types.
  (match-let (((slot-symbol . property) member))
    (let* ((slot (symbol->string slot-symbol))
	   (traits (assoc-with-default '() 'traits property))
	   (name
	    (cond ((assoc '#{smithy.api#xmlName}# traits)
		   => (lambda (pair) (cdr pair)))
		  ((assoc '#{smithy.api#enumValue}# traits)
		   => (lambda (pair) (cdr pair)))
		  (else slot)))
	   (type (assoc-type-of-slot slot property))
	   (required
	    (cond ((assoc '#{smithy.api#required}# traits) #t)
		  (else #f)))
	   ;; (* FLATTENED IS NOT USED. *)
	   (flattened
	    (cond ((assoc '#{smithy.api#xmlFlattened}# traits) #t)
		  (else #f))))
      (cond ((assoc-option '#{smithy.api#httpLabel}# traits)
	     => (lambda (_)
		  (list slot name type 'PATH required)))
	    ((assoc-option '#{smithy.api#httpQuery}# traits)
	     => (lambda (v)
		  (list slot v type 'QUERY required)))
	    ((assoc-option '#{smithy.api#httpHeader}# traits)
	     => (lambda (v)
		  (list slot v type 'HEADER required)))
	    ((assoc-option '#{smithy.api#httpPrefixHeaders}# traits)
	     => (lambda (v)
		  (list slot v type 'HEADER-PREFIX required)))
	    ((assoc-option '#{smithy.api#httpPayload}# traits)
	     => (lambda (_)
		  ;; (* DATA IS CONTENT PAYLOAD. *)
		  (list slot name type 'PAYLOAD required)))
	    (else
	     ;; Empty traits means a response element.
	     (list slot name type 'ELEMENT required))))))

(define (make-composite-type name type-kind members)
  (cons name (cons type-kind (map make-slot-property members))))

(define (make-type-definition shape-element)
  ;; It returns a type definition, consisting of a list (type-name
  ;; type-kind . slot-properties).  A slot-properties describes
  ;; structure slots, when a type-kind is "enum", "list", "structure",
  ;; or "union".  Elements of an enumeration-type have type "Unit", a
  ;; list-type has a single "member" slot, and a map-type has two
  ;; "key" and "value" slots.
  (match-let (((id . property) shape-element))
    (cond
     ((assoc 'type property)
      => (lambda (pair)
	   (let* ((type-kind (cdr pair))
		  (id-string (symbol->string id))
		  (name (drop-prefix "com.amazonaws.s3#" id-string)))
	     (cond
	      ((or (string=? type-kind "operation")
		   (string=? type-kind "service"))
	       ;; Drop "operation" and "service".
	       #f)
	      ((primitive-type? type-kind)
	       (list name type-kind))
	      ((or (string=? type-kind "enum")
		   (string=? type-kind "structure")
		   (string=? type-kind "union"))
	       (let ((members (assoc 'members property)))
		 (assert (not (eqv? members #f)))
		 (make-composite-type name type-kind (cdr members))))
	      ((string=? type-kind "list")
	       (let ((member1 (assoc 'member property)))
		 (assert (not (eqv? member1 #f)))
		 (make-composite-type name type-kind (list member1))))
	      ((string=? type-kind "map")
	       (let* ((key (assoc-option 'key property))
		      (value (assoc-option 'value property))
		      (members (list (cons 'key key) (cons 'value value))))
		 (assert (and (not (eqv? key #f)) (not (eqv? value #f))))
		 (make-composite-type name type-kind members)))
	      (else
	       (format #t "BAD type-kind definition: ~s" shape-element)
	       (values))))))
     (else
      #f))))

(define (check-type-needs-marshaler definition)
  ;; Warns when a structure has a xml-tag specification that differs
  ;; from a name in a structure.  They need custom marshalers.  It
  ;; ignores top-level slots of a request/response structure, because
  ;; they are handled in generated marshalers.
  (match-let (((type-name type-kind . slot-properties) definition))
    (for-each (lambda (property)
		(match-let (((slot name type locus required) property))
		  (when (and (not (string=? slot name)) (eqv? locus #f))
		    (format #t "SLOT TAG NAME DIFFER: ~s~%" property))))
      slot-properties)))

(define (make-type-definition-list shape-elements)
  (let ((definitions (delete #f (map make-type-definition shape-elements))))
    (for-each check-type-needs-marshaler definitions)
    definitions))

;; (make-type-definition (assoc '#{com.amazonaws.s3#AbortIncompleteMultipartUpload}# s3-api))

(define list-of-types (make-type-definition-list s3-api))

;;;
;;; SUMMARY OF ACTIONS
;;;

;; This part makes a catalog of actions in LIST-OF-ACTIONS.  Its
;; entry is a summary of an action: (action-name signature
;; action-property request-properties response-properties).

(define (find-action-structure action-name)
  (let ((key (string-append "com.amazonaws.s3#" action-name)))
    (assoc-option (string->symbol key) s3-api)))

(define (rename-output-structure-name output)
  (string-replace-substring output "Output" "Response"))

#|
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
|#

(define (find-exchange-signature action-structure)
  ;; Returns a request/response name pair ("XXXXRequest"
  ;; "XXXXResponse").  It renames the result type "XXXXOutput" to
  ;; "XXXResponse".  It may return "Unit" for response.  Note the full
  ;; structure names look like: "com.amazonaws.s3#XXXXRequest" and
  ;; "com.amazonaws.s3#XXXXOutput".
  (let ((r1 (assoc-type-of-slot #f (assoc-option 'input action-structure)))
	(q1 (assoc-type-of-slot #f (assoc-option 'output action-structure)))
	(prefix "com.amazonaws.s3#"))
    (assert (and (string? r1) (string? q1)))
    (assert (not (string=? "Unit" r1)))
    (let ((q2 (string-replace-substring q1 "Output" "Response")))
      (list r1 q2))))

#|
(define (find-request-structure~ action-structure)
  (let* ((signature (find-exchange-signature action-structure))
	 (slot-name (string-append "com.amazonaws.s3#" (car signature))))
    (cdr (assoc (string->symbol slot-name) s3-api))))
|#

#|
(define (find-response-structure~ action-structure)
  (let* ((signature (find-exchange-signature action-structure))
	 (slot-name (string-append "com.amazonaws.s3#" (cadr signature))))
    (cdr (assoc (string->symbol slot-name) s3-api))))
|#

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

#|
(define (itemize-slot-properties~ exchange-structure-name)
  ;; Extracts properties of a request/response structure of
  ;; "com.amazonaws.s3#XXXXRequest" and "com.amazonaws.s3#XXXXOutput".
  ;; It returns a list of five-tuples (slot name type locus required), or
  ;; returns an empty list for "Unit".  A slot is a name in a
  ;; request/response strucuture.  A locus indicates where a parameter
  ;; is passed, and it is one of {PATH, QUERY, HEADER,
  ;; HEADER-PREFIX, PAYLOAD, ELEMENT}.  locus=PAYLOAD means the
  ;; value is a whole payload.
  (let* ((prefix "com.amazonaws.s3#")
	 (slot-name (string-append prefix exchange-structure-name))
	 (exchange-structure (assoc-option (string->symbol slot-name) s3-api))
	 (members (assoc-option 'members exchange-structure)))
    (if (eqv? #f members)
	'()
	(delete #f (map make-slot-property members)))))
|#

(define (itemize-slot-properties exchange-structure-name)
  ;; Returns a properties of a request/response structure
  ;; ("XXXXRequest" or "XXXXOutput").  It returns a list of
  ;; five-tuples (slot name type locus required), or an empty list for
  ;; "Unit".
  (cond ((assoc exchange-structure-name list-of-types)
	 => (lambda (definition)
	      (match-let (((type-name type-kind . slot-properties) definition))
		slot-properties)))
	(else
	 '())))

(define (adjust-input-structure-name request)
  (string-replace-substring request "Request" "Input"))

(define (adjust-output-structure-name response)
  (string-replace-substring response "Response" "Output"))

(define (summarize-action action-name)
  ;; Returns a list of (action-name signature action-property
  ;; request-properties response-properties).  A signature is a pair
  ;; of request/response names.  It renames the response name from
  ;; "XXXXOutput" to "XXXResponse".
  (when tr? (format #t ";; summarize-action ~a~%" action-name))
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
;; (summarize-action "AbortMultipartUpload")
;; (summarize-action "DeleteObjects")
;; (summarize-action "UploadPartCopy")

(define list-of-actions (map summarize-action list-of-action-names))

;;;
;;; TYPES APPEARING IN REQUESTS AND ERROR TYPES
;;;

;; This part makes a list of type-names that appear in requests in
;; LIST-OF-TYPES-IN-REQUESTS.

(define (collect-types-in-slots slot-properties acc)
  (if (null? slot-properties)
      acc
      (match-let (((slot name type . _) (car slot-properties)))
	(collect-types-in-slots
	 (cdr slot-properties)
	 (collect-types-in-type type acc)))))

(define (collect-types-in-type type acc)
  ;; Collect types embedded in a type.  It appends new types to acc.
  (cond
   ((string=? "Unit" type)
    acc)
   ((member type acc)
    acc)
   (else
    (let ((acc+ (cons type acc)))
      (match-let* ((definition (assoc type list-of-types))
		   (_ (format #t "collect-types-in-type ~s ~s~%" type definition))
		   ((type-name type-kind . slot-properties) definition))
	(cond ((primitive-type? type-kind)
	       acc+)
	      ;; Composite-types:
	      ((or (string=? type-kind "enum")
		   (string=? type-kind "list")
		   (string=? type-kind "map")
		   (string=? type-kind "structure")
		   (string=? type-kind "union"))
	       (collect-types-in-slots slot-properties acc+))
	      (else
	       (format #t "BAD type-kind ~s~%" type-kind)
	       (values))))))))

(define (collect-types-in-requests request-properties acc)
  (if (null? request-properties)
      acc
      (collect-types-in-requests
       (cdr request-properties)
       (match-let* (((slot name type . _) (car request-properties)))
	 (collect-types-in-type type acc)))))

(define (list-types-in-requests-loop actions acc)
  (if (null? actions)
      acc
      (match-let* (((name signature _ request-properties _) (car actions)))
	(list-types-in-requests-loop
	 (cdr actions)
	 (collect-types-in-requests request-properties acc)))))

(define (list-types-in-requests)
  (sort
   (delete-duplicates
    (list-types-in-requests-loop list-of-actions '()))
   string<?))

;; LIST-ERROR-TYPES returns a list of error types defined in
;; "s3.json".  Errors such as "BucketAlreadyExists" have "traits":
;; {"smithy.api#error": "client", "smithy.api#httpError": 409}.
;;
;; The error types defined are: {"BucketAlreadyExists"
;; "BucketAlreadyOwnedByYou" "EncryptionTypeMismatch"
;; "IdempotencyParameterMismatch" "InvalidObjectState"
;; "InvalidRequest" "InvalidWriteOffset" "NoSuchBucket" "NoSuchKey"
;; "NoSuchUpload" "NotFound" "ObjectAlreadyInActiveTierError"
;; "ObjectNotInActiveTierError" "TooManyParts"}

(define (list-error-types)
  (delete-duplicates
   (apply-append
    (map (lambda (definition)
	   (match-let* (((type-name type-kind . slot-properties) definition))
	     (let* ((slot-name (string-append "com.amazonaws.s3#" type-name))
		    (key (string->symbol slot-name))
		    (type-structure (assoc-option key s3-api))
		    (traits (assoc-option 'traits type-structure))
		    (error-site (assoc-option '#{smithy.api#error}# traits))
		    (error-code (assoc-option '#{smithy.api#error}# traits)))
	       (if error-site
		   (list type-name)
		   '()))))
	 list-of-types))))

(define list-of-types-in-requests (list-types-in-requests))

;;;
;;; PARAMETER INQUERIES
;;;

;; This part makes a list of dispatch entries in COLLECTED-DISPATCHES.
;; A dispatch is a request condition that selects an action.
;; Dispatches are collected to make dispatcher code.

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
	   (format #t "BAD unknown url pattern found: ~s" uri)
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
	  (match-let (((slot name type locus required) (car props)))
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
		  ((HEADER-PREFIX)
		   (format #t "BAD httpPrefixHeaders marked required~%")
		   (values))
		  ((PAYLOAD)
		   (loop (cdr props) queries-acc headers-acc))
		  ((ELEMENT)
		   (loop (cdr props) queries-acc headers-acc))
		  (else
		   (format #t "BAD properties=~s~%" (car props))
		   (values)))))))))

(define (make-dispatch-entry action-name)
  (cond ((assoc action-name list-of-actions)
	 => (lambda (action)
	      (make-request-dispatch action)))
	(else #f)))

;; (make-dispatch-entry "AbortMultipartUpload")
;; (make-dispatch-entry "DeleteObjects")
;; (make-dispatch-entry "UploadPartCopy")

(define (collect-request-dispatches list-of-actions)
  (let loop ((actions list-of-actions)
	     (acc '()))
    (if (null? actions)
	acc
	(let ((dispatch (make-request-dispatch (car actions))))
	  (loop (cdr actions) (append acc (list dispatch)))))))

(define collected-dispatches (collect-request-dispatches list-of-actions))

;;;
;;; DISPATCHER SORTER
;;;

;; This part merges dispatches by a key method-path.  It prepares for
;; registering to an http-server-mux.

(define (merge-request-dispatches list-of-actions)
  ;; Merges request dispatch entries by combining ones with the same
  ;; method-path pair.  It returns an alist with a method-path key and
  ;; a list of dispatches sharing the same key.
  (let loop ((entries collected-dispatches)
	     ;;(entries (collect-request-dispatches list-of-actions))
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

(define merged-dispatches (merge-request-dispatches list-of-actions))
(define list-of-dispatches (sort-dispatches merged-dispatches))

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
	(string-append "(" (string-join v " && ") ")"))))

(define (make-choice-clause dispatch)
  (match-let* (((name _ queries headers signature) dispatch)
	       (q (make-conditionals (append queries headers)))
	       (body (string-append "h_" name "(bbs, w, r)")))
    (format #f "if ~a {~a}" q body)))

(define (list-queries-headers dispatches)
  ;; Gathers queries and headers from dispatches.  The result is used
  ;; to access queries/headers in the dispatcher code.
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
   (list "// dispatcher.go (2025-10-01)"
	 "// Dispatcher for net/http.ServeMux.  It switches handlers"
	 "// with regard to method-path patterns and required"
	 "// parameters in request API."
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

(define (write-dispatcher port)
  (let ((ss (generate-dispatcher list-of-dispatches)))
    (format port "~a" (string-join ss "\n"))))

(define (display-dispatcher)
  (write-dispatcher #t))

(define (dump-dispatcher file)
  (call-with-output-file file
    (lambda (port)
      (write-dispatcher port))))

;;;
;;; HANDLER PRINTER
;;;

;; This part prints handler functions, which are used in the
;; dispatcher.

(define (locus-ordered? property-a property-b)
  (match-let (((slot-a name-a type-a locus-a required-a) property-a)
	      ((slot-b name-b type-b locus-b required-b) property-b))
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

(define (make-sdk-enumerator type-name enumerator-string)
  ;; Makes an enumerator of an enum of AWS-SDK.
  (format #f "types.~a~a" type-name (camelcase-string enumerator-string)))

(define (make-enumerator-importer-function type slot-properties)
  ;; A slot property of an enumeration-type is (slot name "Unit" ...),
  ;; and a name is an enumerator string representation.
  (append
   (list (format #f "func import_~a(s string) (types.~a, error) {" type type)
	 (format #f "switch s {"))
   (map (lambda (property)
	  (match-let (((slot name _ . _) property))
	    (let ((enumerator (make-sdk-enumerator type name)))
	      (format #f "case ~s: return ~a, nil" name enumerator))))
	slot-properties)
   (list "default: var err1 = errors.New(\"interning an enum\")"
	 "log.Fatal(err1); return \"\", err1}}")))

(define (make-enumerator-importers list-of-types-in-requests)
  ;; Makes importer functions for enumerators.  Functions are named
  ;; with an enumeration name prefixed by "import_".
  (apply-append
   (map (lambda (type)
	  (match-let* ((definition (assoc type list-of-types))
		       ((type-name type-kind . slot-properties) definition))
	    (if (string=? type-kind "enum")
		(make-enumerator-importer-function type slot-properties)
		'())))
	list-of-types-in-requests)))

(define (make-coercing-import type-name assigner rhs)
  ;; Makes a coercion of a string to a given type.  Calling an
  ;; assigner makes an assignment.  It assumes a type-name is defined.
  (when tr? (format #t ";; . make-coercing-import ~s~%" type-name))
  (match-let* ((definition (assoc type-name list-of-types))
	       ((type-name type-kind . slot-properties) definition))
    (cond
     ;; Primitive-types:
     ((string=? type-kind "blob")
      (values))
     ((string=? type-kind "boolean")
      (list
       (format #f "{var x, err2 = strconv.ParseBool(~a)" rhs)
       "if err2 != nil {log.Fatal(err2); return err2}"
       (string-append (assigner "&x") "}")))
     ((string=? type-kind "integer")
      (list
       (format #f "{var x1, err2 = strconv.ParseInt(~a, 10, 32)" rhs)
       "if err2 != nil {log.Fatal(err2); return err2}"
       "var x2 = int32(x1)"
       (string-append (assigner "&x2") "}")))
     ((string=? type-kind "long")
      (list
       (format #f "{var x, err2 = strconv.ParseInt(~a, 10, 64)" rhs)
       "if err2 != nil {log.Fatal(err2); return err2}"
       (string-append (assigner "&x") "}")))
     ((string=? type-kind "operation")
      (values))
     ((string=? type-kind "service")
      (values))
     ((string=? type-kind "string")
      (list
       (assigner (format #f "thing_pointer(~a)" rhs))))
     ((string=? type-kind "timestamp")
      (list
       (format #f "{var x, err2 = time.Parse(time.RFC3339, ~a)" rhs)
       "if err2 != nil {log.Fatal(err2); return err2}"
       (string-append (assigner "&x") "}")))
     ;; Composite-types:
     ((string=? type-kind "enum")
      (list
       (format #f "{var x, err2 = import_~a(~a)" type-name rhs)
       "if err2 != nil {log.Fatal(err2); return err2}"
       (string-append (assigner "x") "}")))
     ((string=? type-kind "list")
      (list
       (assigner (format #f "~a" rhs))))
     ((string=? type-kind "map")
      ;; Map is handled by a caller.
      '())
     ((string=? type-kind "structure")
      (list
       (assigner (format #f "~a" rhs))))
     ((string=? type-kind "union")
      (list
       (assigner (format #f "~a" rhs))))
     ;; Others:
     (else
      (format #t "BAD type-kind ~s~%" type-kind)
      (values)))))

(define (make-coercing-export type-name assigner rhs)
  ;; Makes a coercion of a given type to a string.  It is in the
  ;; opposite direction of make-extern-coercing.  It errs when a
  ;; type-name is not defined.
  ;;-(format #t "make-coercing-export ~s ~s~%" type-name (assoc type-name list-of-types))
  (match-let* ((definition (assoc type-name list-of-types))
	       ((_ type-kind . slot-properties) definition))
    (cond
     ;; Primitive-types:
     ((string=? type-kind "blob")
      (values))
     ((string=? type-kind "boolean")
      (list
       (assigner (format #f "strconv.FormatBool(*~a)" rhs))))
     ((string=? type-kind "integer")
      (list
       (assigner (format #f "strconv.FormatInt(int64(*~a), 10)" rhs))))
     ((string=? type-kind "long")
      (list
       (assigner (format #f "strconv.FormatInt(*~a, 10)" rhs))))
     ((string=? type-kind "operation")
      (values))
     ((string=? type-kind "service")
      (values))
     ((string=? type-kind "string")
      (list
       (assigner (format #f "string(*~a)" rhs))))
     ((string=? type-kind "timestamp")
      (list
       (assigner (format #f "~a.String()" rhs))))
     ;; Composite-types:
     ((string=? type-kind "enum")
      (list
       (assigner (format #f "string(~a)" rhs))))
     ((string=? type-kind "list")
      (list
       (assigner (format #f "~a" rhs))))
     ((string=? type-kind "map")
      ;; Map is handled by a caller.
      '())
     ((string=? type-kind "structure")
      (list
       (assigner (format #f "~a" rhs))))
     ((string=? type-kind "union")
      (list
       (assigner (format #f "~a" rhs))))
     (else
      (format #t "BAD type-kind ~s~%" type-kind)
      (values)))))

(define (resolve-type type-name)
  ;; Returns a type representation.  Primitive types resolve to a type
  ;; name in Golang.  Composite types resolve to itself that should be
  ;; defined types.
  (format #t ";; . resolve-type ~s ~s~%" type-name
	  (assoc type-name list-of-types))
  (match-let* ((definition (assoc type-name list-of-types))
	       ((_ type-kind . slot-properties) definition))
    (if (primitive-type? type-kind)
	type-kind
	(format #f "types.~a" type-name))))

(define (make-input-assignment request-property)
  ;; Makes an assignment in a structure "s3.XXXXInput" of AWS-SDK.
  ;; Slot property is a list of five-tuples (slot name type locus
  ;; required).  Note the structure name of a request is "XXXXRequest"
  ;; in the API and Smithy.
  (when tr? (format #t ";; . make-input-assignment1 ~s~%" (list-ref request-property 2)))
  (match-let* (((slot name type locus required) request-property)
	       (definition (assoc type list-of-types))
	       ((type-name type-kind . slot-properties) definition))
    (when tr? (format #t ";; . make-input-assignment2 ~s~%" definition))
    (case locus
      ((PATH)
       ;; Ignore path parameters.
       '())
      ((QUERY)
       ;; (list (format #f "i.~a = qi.Get(~s)" slot name))
       (let ((rhs (format #f "qi.Get(~s)" name))
	     (assigner (lambda (rhs)
			 (format #f "i.~a = ~a" slot rhs))))
	 (make-coercing-import type assigner rhs)))
      ((HEADER)
       ;; (list (format #f "i.~a = hi.Get(~s)" slot name))
       (cond
	((string=? type-kind "map")
	 (format #t "AHO MAP ~s slot-properties=~s~%" name slot-properties)
	 ;; IT ASSUMES ONLY STRING MAPS.
	 (match-let ((((_ _ type2 . _) (_ _ type3 . _)) slot-properties))
	   (let* ((key-type (resolve-type type2))
		  (value-type (resolve-type type3))
		  (_ (assert (string=? key-type "string")))
		  (_ (assert (string=? value-type "string")))
		  (map-type (format #f "map[~a]~a" key-type value-type))
		  (assigner (lambda (rhs)
			      (format #f "bin[key] = ~a" rhs))))
	     (list (format #f "{var rhs = hi.Get(~s)" name)
		   (format #f "var bin ~a" map-type)
		   (format #f "maps.All(rhs)(func (k, v string) bool {")
		   ;;(make-coercing-import type3 assigner "val")
		   (format #f "bin[k] = v}")
		   (format #f "i.~a = bin}" slot)))))
	((string=? type-kind "list")
	 ;; List's slot-properties is (("member" "member" type2 . _))
	 (format #t "AHO LIST slot-properties=~s~%" slot-properties)
	 (match-let (((_ _ type2 . _) (car slot-properties)))
	   (let* ((element-type (resolve-type type2))
		  (assigner (lambda (rhs)
			      (format #f "bin = append(bin, ~a)" rhs))))
	     (append
	      (list (format #f "{var rhs = hi.Values(~s)" name)
		    (format #f "var bin []~a" element-type)
		    (format #f "for _, v := range slices.All(rhs) {"))
	      (make-coercing-import type2 assigner "v")
	      (list ;;(format #f "bin = append(bin, v)}")
	       (format #f "}")
	       (format #f "i.~a = bin}" slot))))
	   ))
	(else
	 (let ((rhs (format #f "hi.Get(~s)" name))
	       (assigner (lambda (rhs)
			   (format #f "i.~a = ~a" slot rhs))))
	   (make-coercing-import type assigner rhs)))))
      ((HEADER-PREFIX)
       ;; IT ASSUMES ONLY STRING MAPS.
       (assert (string=? type-kind "map"))
       (match-let ((((_ _ type2 . _) (_ _ type3 . _)) slot-properties))
	 (let* ((key-type (resolve-type type2))
		(value-type (resolve-type type3))
		(_ (assert (string=? key-type "string")))
		(_ (assert (string=? value-type "string")))
		(map-type (format #f "map[~a]~a" key-type value-type))
		(assigner (lambda (rhs)
			    (format #f "bin[key] = ~a" rhs))))
	   (list (format #f "{var prefix = http.CanonicalHeaderKey(~s)" name)
		 (format #f "var bin ~a" map-type)
		 ;; (format #f "var bin map[string]string")
		 (format #f "for k, v := range hi {")
		 (format #f "if strings.HasPrefix(k, prefix) {bin[k] = v[0]}}")
		 (format #f "i.~a = bin}" slot)))))
      ((PAYLOAD)
       (cond
	((string=? type-kind "blob")
	 ;; Ignore blob.
	 '())
	(else
	 ;; Payload types are: {CompletedMultipartUpload,
	 ;; CreateBucketConfiguration, Delete, Tagging}.
	 (list
	  (format #f "{var x types.~a" type)
	  "var bs, err1 = io.ReadAll(r.Body)"
	  "if err1 != nil {log.Fatal(err1); return err1}"
	  "var err2 = xml.Unmarshal(bs, &x)"
	  "if err2 != nil {log.Fatal(err2); return err2}"
	  (format #f "i.~a = &x}" slot)))))
      ((ELEMENT)
       (format #t "BAD properties=~s~%" request-property)
       (values))
      (else
       (format #t "BAD properties=~s~%" request-property)
       (values)))))

(define (make-output-extraction response-property)
  ;; Makes extraction code from structure "s3.XXXXOutput" of AWS SDK.
  ;; Each property is a list of five-tuples (slot name type locus
  ;; required).
  (match-let* (((slot name type locus required) response-property)
	       (definition (assoc type list-of-types))
	       ((type-name type-kind . slot-properties) definition))
    (when tr? (format #t ";; . make-output-extraction ~s~%" slot))
    (case locus
      ((PATH)
       '())
      ((QUERY)
       (begin
	 (format #t "BAD query in response: ~s~%" name)
	 (values)))
      ((HEADER HEADER-PREFIX)
       ;;(list (format #f "ho.Add(~s, s.~a)" name slot))
       (cond
	((string=? type-kind "map")
	 (let ((rhs (format #f "s.~a" slot))
	       (assigner (lambda (rhs)
			   (format #f "ho.Add(~s, ~a)" name rhs))))
	   (make-coercing-export type assigner rhs)))
	((string=? type-kind "list")
	 (let ((rhs (format #f "s.~a" slot))
	       (assigner (lambda (rhs)
			   (format #f "ho.Add(~s, ~a)" name rhs))))
	   (make-coercing-export type assigner rhs)))
	(else
	 (let ((rhs (format #f "s.~a" slot))
	       (assigner (lambda (rhs)
			   (format #f "ho.Add(~s, ~a)" name rhs))))
	   (make-coercing-export type assigner rhs)))))
      ((PAYLOAD)
       (begin
	 (when tr? (format #t ";; Payload in response: ~s~%" name))
	 '()))
      ((ELEMENT)
       (begin
	 ;; (when tr? (format #t ";; Skip element in response: ~s~%" name))
	 '()))
      (else
       (format #t "BAD properties=~s~%" response-property)
       (values)))))

(define (make-output-payload-extraction just-status code)
  (append
   (list
    "ho.Set(\"Content-Type\", \"application/xml\")")
   (if (not just-status)
       (list
	"var co, err5 = xml.MarshalIndent(s, \" \", \"  \")"
	"if err5 != nil {log.Fatal(err5); return err5}")
       '())
  (list
   (format #f "var status int = ~a" code)
   "w.WriteHeader(status)")
  (if (not just-status)
      (list
       "var _, err6 = w.Write(co)"
       "if err6 != nil {log.Fatal(err6); return err6}")
      '())))

(define (make-handler-function action)
  (match-let*
      (((name signature action-property request-properties response-properties)
	action)
       ((request-name response-name) signature)
       (input-name (adjust-input-structure-name request-name))
       (output-name (adjust-output-structure-name response-name))
       (properties (cast-payload-property-rear request-properties))
       ((_ _ code) action-property))
    (when tr? (format #t ";; make-handler-function ~s~%" name))
    (append
     (list
      ;; Start of function declaration:
      (string-append (format #f "func h_~a" name)
		     "(bbs *BB_server,"
		     " w http.ResponseWriter, r *http.Request) error {")
      "var qi = r.URL.Query()"
      "var hi = r.Header"
      "var ho = w.Header()"
      "// Mark variables used to avoid unused errors:"
      "var _, _, _ = qi, hi, ho")
     ;; Input accessors:
     (list
      (format #f "var i = s3.~a{}" input-name))
     (apply append (map make-input-assignment properties))
     ;; Hander invocation:
     (list
      "var ctx = r.Context()"
      (if (string=? output-name "Unit")
	  (format #f "var _, err3 = bbs.~a(ctx, &i)" name)
	  (format #f "var o, err3 = bbs.~a(ctx, &i)" name))
      "if err3 != nil {log.Fatal(err3); return err3}")
     ;; Output accessors:
     (if (string=? output-name "Unit")
	 (make-output-payload-extraction #t code)
	 (append
	  (list
	   (format #f "var s = s_~a(*o)" response-name))
	  (apply append (map make-output-extraction response-properties))
	  (make-output-payload-extraction #f code)))
     ;; Function end:
     (list "return nil}"))))

(define (display-handler-function action)
  (let ((ss (make-handler-function action)))
    (format #t "~a~%" (apply string-append (intervene-separator "\n" ss)))))

(define (make-auxiliary-functions)
  (list "func gather_headers(h http.Header, prefix string) map[string]string {"
	"var p = http.CanonicalHeaderKey(prefix)"
	"var m map[string]string"
	"for k, v := range h {"
	"if strings.HasPrefix(k, p) {"
	"m[k] = v[0]}}"
	"return m}"))

(define (make-handler-file list-of-actions)
  (append
   (list "// handlers.go (2025-10-01)"
	 "// API-STUB.  Handler functions (h_XXXX) called from the"
	 "// dispatcher."
	 "package server"
	 "import ("
	 ;; "\"context\""
	 "\"encoding/xml\""
	 "\"errors\""
	 "\"io\""
	 "\"log\""
	 "\"net/http\""
	 "\"slices\""
	 "\"strings\""
	 "\"strconv\""
	 "\"time\""
	 "\"github.com/aws/aws-sdk-go-v2/service/s3\""
	 "\"github.com/aws/aws-sdk-go-v2/service/s3/types\""
	 ")")
   (apply append
	  (map make-handler-function list-of-actions))))

(define (write-handlers port)
  (let ((ss (make-handler-file list-of-actions)))
    (format port "~a~%" (apply string-append (intervene-separator "\n" ss)))))

(define (display-handlers)
  (write-handlers #t list-of-actions))

(define (write-enumerator-importers port)
  (let ((ss (make-enumerator-importers list-of-types-in-requests)))
    (format port "~a~%" (apply string-append (intervene-separator "\n" ss)))))

(define (write-auxiliary-functions port)
  (let ((ss (make-auxiliary-functions)))
    (format port "~a~%" (apply string-append (intervene-separator "\n" ss)))))

(define (dump-handlers file)
  (call-with-output-file file
    (lambda (port)
      (write-handlers port)
      (write-enumerator-importers port)
      (write-auxiliary-functions port))))

;; (display-handler-function (assoc "ListParts" list-of-actions))

;;;
;;; RESPONSE MARSHALER PRINTER
;;;

#|
(define (make-response-marshaler-preamble~)
  (list
   (format #f "func p[T any](v T) *T {return &v}")
   (format #f "func start_element(k string) xml.StartElement {~a}"
	   "return xml.StartElement{Name: xml.Name{Local: k}")))
|#

(define (check-output-in-payload response-properties)
  ;; Checks if response is in a payload.  "CopyObject" has
  ;; "CopyObjectResult", "GetObject" has "Body", and "UploadPartCopy"
  ;; has "CopyPartResult"
  (let ((check-output-in-payload1
	 (lambda (property)
	   (match-let (((slot name type locus required) property))
	     (eqv? locus 'PAYLOAD)))))
    (any check-output-in-payload1 response-properties)))

(define (make-slot-marshaler property)
  ;; Returns a list of marshaler lines of an response element.
  (match-let (((slot name type locus required) property))
    (case locus
      ((PATH QUERY)
       (format #t "BAD property in response: ~s~%" property)
       '())
      ((PAYLOAD)
       (list
	(format #f "{var err2 = e.EncodeElement(s.~a, start_element(\"~a\"))" slot name)
	(format #f "if err2 != nil {return err2}}")))
      ((HEADER HEADER-PREFIX)
       '())
      ((ELEMENT)
       (list
	(format #f "{var err2 = e.EncodeElement(s.~a, start_element(\"~a\"))" slot name)
	(format #f "if err2 != nil {return err2}}")))
      (else
       (format #t "BAD property in response: ~s~%" property)
       '()))))

(define (make-marshaler-function action)
  ;; Returns lines of a response marshaler for "XXXXResponse".
  (match-let*
      (((name (request-name response-name) _ _ response-properties) action)
       (output-name (adjust-output-structure-name response-name))
       (output-in-payload (check-output-in-payload response-properties))
       (encoders (delete '() (map make-slot-marshaler response-properties)))
       (nothing-in-payload (= (length encoders) 0)))
    (when tr? (format #t ";; make-marshaler-function ~s~%" name))
    (assert (or (not output-in-payload) (= (length encoders) 1)))
    (if (string=? output-name "Unit")
	'()
	(append
	 (list
	  (format #f "type s_~a s3.~a" response-name output-name)
	  (format #f "func (s s_~a) MarshalXML~a error {"
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
	  (format #f "return nil}"))))))

(define (display-repsonse-marshaler1~ action-name)
  (let* ((action (assoc action-name list-of-actions))
	 (ss (make-marshaler-function action))
	 (lines (apply string-append (intervene-separator "\n" ss))))
    (format #t "~a~%" lines)
    (values)))

(define (display-repsonse-marshaler~)
  (let ((s1 (make-response-marshaler-preamble~))
	(s2 (apply append (map make-marshaler-function list-of-actions))))
    (format #t "~a~%~a~%"
	    (apply string-append (intervene-separator "\n" s1))
	    (apply string-append (intervene-separator "\n" s2)))))

(define (make-marshaler-file list-of-actions)
  (append
   (list "// marshalers.go (2025-10-01)"
	 "// API-STUB.  Marshalers of response structures.  Response"
	 "// structures need custom marshalers, because they have"
	 "// some slots that need to be renamed and also have an"
	 "// extra slot that should be suppressed."
	 "package server"
	 "import ("
	 ;; "\"context\""
	 "\"encoding/xml\""
	 "\"github.com/aws/aws-sdk-go-v2/service/s3\""
	 ")"
	 "func thing_pointer[T any](v T) *T {return &v}"
	 "func start_element(k string) xml.StartElement {"
	 "return xml.StartElement{Name: xml.Name{Local: k}}"
	 "}")
   (apply append
	  (map make-marshaler-function list-of-actions))))

(define (write-marshalers port list-of-actions)
  (let ((ss (make-marshaler-file list-of-actions)))
    (format port "~a~%" (apply string-append (intervene-separator "\n" ss)))))

(define (display-marshalers)
  (write-marshalers #t list-of-actions))

(define (dump-marshalers file)
  (call-with-output-file file
    (lambda (port)
      (write-marshalers port list-of-actions))))

;; (make-marshaler-function (assoc "CopyObject" list-of-actions))
;; (display-repsonse-marshaler)
;; (display-marshaler-function (assoc "ListParts" list-of-actions))

;;;
;;; SERVER TEMPLATE PRINTER
;;;

(define (make-api-template action)
  (match-let* (((name signature action-property _ _) action)
	       ((request-name response-name) signature)
	       (input-name (adjust-input-structure-name request-name))
	       (output-name (adjust-output-structure-name response-name)))
    (when tr? (format #t ";; make-api-template ~s~%" name))
    (let ((api-output-name
	   (if (string=? output-name "Unit")
	       (format #f "~aOutput" name)
	       output-name)))
      (list
       (string-append
	(format #f "func (bbs *BB_server) ~a" name)
	(format #f "(ctx context.Context, params *s3.~a," input-name)
	(format #f " optFns ...func(*s3.Options))")
	(format #f " (*s3.~a, error) {" api-output-name))
       (format #f "var o = s3.~a{}" api-output-name)
       "return &o, nil}"))))

(define (make-api-template-file list-of-actions)
  (append
   (list "// template.go (2025-10-01)"
	 "// API-STUB.  Handler templates. They should be replaced by"
	 "// actual implementations."
	 "package server"
	 "import ("
	 "\"context\""
	 "\"github.com/aws/aws-sdk-go-v2/service/s3\""
	 ")")
   (apply append
	  (map make-api-template list-of-actions))))

(define (write-api-template-file port list-of-actions)
  (let ((ss (make-api-template-file list-of-actions)))
    (format port "~a~%" (apply string-append (intervene-separator "\n" ss)))))

(define (dump-template file)
  (call-with-output-file file
    (lambda (port)
      (write-api-template-file port list-of-actions))))

;; (dump-dispatcher "dispatcher.go")
;; (dump-handlers "handlers.go")
;; (dump-marshalers "marshalers.go")
;; (dump-template "api-template.go")
