;; stub.scm (2025-10-16)
;; Copyright 2025-2025 RIKEN R-CCS.
;; SPDX-License-Identifier: BSD-2-Clause

;; Ad-hoc server stub generator.  This generates dispatcher code for
;; AwS-S3 requests.  It reads "s3.json" in Smithy-2.0 and generates
;; dispatcher code.

;; This is for "guile --r7rs" and tested with GNU-Guile-3.0.10.

;; It generates files "api-template.go", "dispatcher.go",
;; "handler.go", and "marshaler.go".  A dispatcher in
;; "dispatcher.go" is a request multiplexer.  It calls routines in
;; "handler.go" to process requests and responses.  Marshalers in
;; "marshaler.go" are needed to skip an extra memeber in responses,
;; because the output structures in AWS-SDK have one added member.
;; "api-template.go" is a skeleton code of API handlers.  The files
;; other than "api-template.go" should be placed in the same package.

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

;; Note on AWS-SDK.  Input parameter "XXXXRequest" in API (and Smithy)
;; has a name "XXXXInput" in AWS-SDK.

;; ASSUMPTION: It assumes enumeration types are string types.  Null
;; value for enumeration types is "".

;; NOTE: Camelcase conversion makes enumeration "ETag" of
;; "ObjectAttributes" is converted to "ObjectAttributesEtag" in
;; AWS-SDK ("ETag" to "Etag", "t" in lowercase).

;; NOTE: It ignores "smithy.rules#contextParam" which indicates
;; "Bucket" name is given by host-name part.

;; Golang http server: https://pkg.go.dev/net/http#ServeMux

(import
 (ice-9 exceptions)
 (ice-9 binary-ports)
 (ice-9 textual-ports)
 (ice-9 expect)
 (ice-9 popen)
 (ice-9 format)
 (ice-9 match)
 (ice-9 regex)
 (ice-9 string-fun) ;; string-replace-substring
 ;;(scheme base)
 ;;(srfi srfi-133) ;; r7rs-vector-library (NO srfi-133 in Guile)
 (only (scheme base) define-record-type)
 (only (scheme base) vector-map vector-for-each vector->list)
 (only (scheme base) write-string textual-port? read-error?)
 (srfi srfi-1) ;; list
 (srfi srfi-11) ;; multiple-values
 (srfi srfi-19) ;; current-date, date->string
 (srfi srfi-60) ;; arithmetic-shift
 (only (rnrs base) infinite? assert))

(setlocale LC_ALL "C.utf-8")

(define (assume . bs) '())
(define (%read-error? x)
  (read-error? x))
(define (valid-number? string)
  (number? (string->number string)))

(load "../test/minima/srfi-180-body.scm")

;; Package path/name of s3-baby-server.  One generated file
;; "api-template.go" shall be placed in bb-server-package, and other
;; generated files "dispatcher.go", "handler.go", and "marshaler.go"
;; in bb-dispatcher-package.

(define bb-package-path "s3-baby-server/internal")
(define bb-dispatcher-package "server")
(define bb-server-package "server")
(define bb-server-name "Bb_server")
(define bb-server-type
  (if (not (string=? bb-server-package bb-dispatcher-package))
      (string-append bb-server-package "." bb-server-name)
      bb-server-name))

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

(define generation-date (date->string (current-date) "~1"))

(define tr? #t)

(define apply-append concatenate)
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
       (apply-append (map (lambda (s) (string-split s (car chars))) strings))
       (cdr chars))))

(define (camelcase-edges s start indices)
  ;; (First call this with (camelcase-edges string 0 '())).
  (cond ((string-match "[a-z][A-Z]" s start)
	 => (lambda (m)
	      (let ((position (+ 1 (car (vector-ref m 1)))))
		(camelcase-edges
		 s position (append  indices (list position))))))
	(else
	 indices)))

(define (camelcase-split s)
  ;; Splits a string at the edges of camelcase.  "camelCASe" to "caml"
  ;; and "CASe".
  (let ((indices (camelcase-edges s 0 '())))
    (map (lambda (b e)
	   (substring s b e))
	 (append '(0) indices)
	 (append indices (list (string-length s))))))

(define (capitalize-string s)
  ;; Capitalizes the string.  It accepts zero-length strings.
  (if (string-null? s)
      s
      (string-append (string (char-upcase (string-ref s 0)))
		     (string-downcase (substring s 1)))))

(define (camelcase-string s)
  ;; Makes a string in camelcase: "baby_server", "BABY_SERVER", and
  ;; "BabyServer" to "BabyServer".
  ;;
  ;; PARTICULAR CASES OF CAMELCASE CONVERSION:
  ;; ObjectAttributes + ETag => ObjectAttributesEtag
  ;; ChecksumAlgorithm + CRC32C => ChecksumAlgorithmCrc32c
  ;; ChecksumAlgorithm + CRC64NVM => ChecksumAlgorithmCrc64nvme
  (let* ((tokens1 (string-split-n (list s) '(#\_ #\- #\:)))
	 (tokens2 (apply-append (map camelcase-split tokens1))))
    (apply string-append (map capitalize-string tokens2))))

(define (string-append-on-tail list s)
  ;; Calls string-append with s on the last element of the list.
  (foldr (lambda (a b) (if (eqv? b s)
			   (cons (string-append a s) '())
			   (cons a b)))
	 s
	 list))

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

;; This part makes a list of type-definitions, which are stored in
;; LIST-OF-TYPES.

;; A TYPE-DEFINITION is a three-tuple plus a list (type-name type-kind
;; tag . slot-property ...).  A TAG is an xml-tag used in marshaling.
;; It is one cited by "smithy.api#xmlName" or #f otherwise.  Each
;; SLOT-PROPERTY describes record slots, when a TYPE-KIND is "enum",
;; "list", "map", "structure", or "union".  An enumeration type has
;; type "Unit" in the elements of it.  A list type has a single
;; "member" slot.  A map type has two "key" and "value" slots.

;; A SLOT-PROPERTY is a five-tuple: (slot tag type locus required).
;; It describes slots of a composite type.  A SLOT is a name of a
;; record slot (it is specified by a key part in Smithy).  A TAG is
;; either a key in queries/headers, an xml-tag (specified by
;; "smithy.api#xmlName"), or an enumerator in enumeration types
;; (specified by "smithy.api#enumValue").  A tag is used as an xml-tag
;; when it is marshaled.  A TYPE is a type-name of this slot
;; (specified by "target").  A LOCUS indicates where a parameter is
;; passed, and it is one of {PATH, QUERY, HEADER, HEADER-PREFIX,
;; PAYLOAD, ELEMENT}.  locus=PAYLOAD means the value is a whole
;; payload.  A locus and a required are only meaningful in
;; request/response structures.

;; Types in AWS-S3 are all named, and they are of a type-kind in:
;; {"blob", "boolean", "enum"*, "integer", "list"*, "long", "map"*,
;; ("operation"), ("service"), "string", "structure"*, "timestamp",
;; "union"*, "Unit"}.
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

(define (assoc-tag-of-slot default property)
  ;; Returns a tag of the slot which is specified in "traits".  It
  ;; accepts #f for property alist.
  (let ((traits (assoc-with-default '() 'traits property)))
    (cond ((assoc '#{smithy.api#xmlName}# traits)
	   => (lambda (pair) (cdr pair)))
	  ((assoc '#{smithy.api#enumValue}# traits)
	   => (lambda (pair) (cdr pair)))
	  ((assoc '#{smithy.rules#contextParam}# traits)
	   => (lambda (_)
		;; (format #t ";; Ignore smithy.rules#contextParam~%")
		default))
	  (else default))))

(define (make-slot-property member)
  ;; Admits an element of "members" and returns a five-tuple (slot tag
  ;; type locus required), describing a record slot of a composite
  ;; type.
  (match-let (((slot-symbol . property) member))
    (let* ((slot (symbol->string slot-symbol))
	   (traits (assoc-with-default '() 'traits property))
	   (tag (assoc-tag-of-slot #f property))
	   (type (assoc-type-of-slot slot property))
	   (required (cond ((assoc '#{smithy.api#required}# traits) #t)
			   (else #f)))
	   ;; (* FLATTENED IS NOT USED. *)
	   (flattened
	    (cond ((assoc '#{smithy.api#xmlFlattened}# traits) #t)
		  (else #f))))
      (cond ((assoc-option '#{smithy.api#httpLabel}# traits)
	     => (lambda (_)
		  (list slot tag type 'PATH required)))
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
		  (list slot tag type 'PAYLOAD required)))
	    (else
	     ;; Empty traits means a response element.
	     (list slot tag type 'ELEMENT required))))))

(define (make-primitve-type type-name type-kind tag)
  (list type-name type-kind tag))

(define (make-composite-type type-name type-kind tag members)
  (let ((tuple (list type-name type-kind tag)))
    (append tuple (map make-slot-property members))))

(define (make-type-definition shape-element)
  ;; Makes a type-definition from an element in "shape" in "s3.json".
  ;; It returns #f when a shape-element is not a type.  A
  ;; type-definition is a three-tuple plus a list (type-name type-kind
  ;; tag . slot-property ...).
  (match-let (((id . property) shape-element))
    (cond
     ((assoc 'type property)
      => (lambda (pair)
	   (let* ((type-kind (cdr pair))
		  (id-string (symbol->string id))
		  (type-name (drop-prefix "com.amazonaws.s3#" id-string))
		  (tag (assoc-tag-of-slot #f property)))
	     (cond
	      ((or (string=? type-kind "operation")
		   (string=? type-kind "service"))
	       ;; Drop "operation" and "service".
	       #f)
	      ((primitive-type? type-kind)
	       (make-primitve-type type-name type-kind tag))
	      ((or (string=? type-kind "enum")
		   (string=? type-kind "structure")
		   (string=? type-kind "union"))
	       (let ((members (assoc 'members property)))
		 (assert (not (eqv? members #f)))
		 (make-composite-type type-name type-kind tag (cdr members))))
	      ((string=? type-kind "list")
	       (let ((member1 (assoc 'member property)))
		 (assert (not (eqv? member1 #f)))
		 (make-composite-type type-name type-kind tag (list member1))))
	      ((string=? type-kind "map")
	       (let* ((key (assoc-option 'key property))
		      (value (assoc-option 'value property))
		      (members (list (cons 'key key) (cons 'value value))))
		 (assert (and (not (eqv? key #f)) (not (eqv? value #f))))
		 (make-composite-type type-name type-kind tag members)))
	      (else
	       (format #t "BAD type-kind definition: ~s" shape-element)
	       (error "BAD type-kind definition" shape-element))))))
     (else
      #f))))

(define (assert-type-needs-no-marshaler definition)
  ;; Warns when a type has a xml-tag specification that differs from a
  ;; name in a record.  It needs custom marshalers when it has
  ;; different names.  It ignores top-level slots of a
  ;; request/response records, because they are handled in
  ;; marshalers of the stub-generator.
  (match-let (((type-name type-kind tag . slot-properties) definition))
    (for-each (lambda (property)
		(match-let (((slot tag type locus required) property))
		  (when (and (not (eqv? #f tag))
			     (eqv? locus #f)
			     (not (string=? slot tag)) )
		    (format #t "SLOT TAG NAME DIFFER: ~s~%" property)
		    (error "SLOT TAG NAME DIFFER" property))))
      slot-properties)))

(define (make-type-definition-list shape-elements)
  (let ((definitions (delete #f (map make-type-definition shape-elements))))
    (for-each assert-type-needs-no-marshaler definitions)
    definitions))

;; (make-type-definition (assoc '#{com.amazonaws.s3#AbortIncompleteMultipartUpload}# s3-api))

(define list-of-types (make-type-definition-list s3-api))

;;;
;;; ACTION SUMMARIZER
;;;

;; This part makes a catalog of actions in LIST-OF-ACTIONS.  Its entry
;; is a summary of an action.

;; An ACTION is a five-tuple (action-name signature action-property
;; request-properties response-properties).  A SIGNATURE is a
;; two-tuple of (request-type response-type).  An ACTION-PROPERTY is a
;; three-tuple (method uri code).  It consists of a method type, a uri
;; path pattern, and an http status code for a successful response.  A
;; REQUEST-PROPERTY and a RESPONSE-PROPERTIES are a list of
;; slot-properties.

(define (adjust-input-structure-name request)
  (string-replace-substring request "Request" "Input"))

(define (adjust-output-structure-name response)
  (string-replace-substring response "Response" "Output"))

(define (rename-output-structure-name~ output)
  (string-replace-substring output "Output" "Response"))

(define (find-action-structure action-name)
  (let ((key (string-append "com.amazonaws.s3#" action-name)))
    (assoc-option (string->symbol key) s3-api)))

(define (make-exchange-signature action-structure)
  ;; Returns a request/response name pair ("XXXXRequest"
  ;; "XXXXResponse").  It renames the result type name "Output" to
  ;; "Response".  It may return "Unit" for response.  Note the full
  ;; structure names look like: "com.amazonaws.s3#XXXXRequest" and
  ;; "com.amazonaws.s3#XXXXOutput".
  (let ((r1 (assoc-type-of-slot #f (assoc-option 'input action-structure)))
	(q1 (assoc-type-of-slot #f (assoc-option 'output action-structure)))
	(prefix "com.amazonaws.s3#"))
    (assert (and (string? r1) (string? q1)))
    (assert (not (string=? "Unit" r1)))
    (let ((q2 (string-replace-substring q1 "Output" "Response")))
      (list r1 q2))))

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

(define (itemize-slot-properties exchange-structure-name)
  ;; Returns a properties of a request/response structure
  ;; ("XXXXRequest" or "XXXXOutput").  It returns a list of
  ;; slot-properties, each of which is a five-tuple (slot tag type
  ;; locus required), or an empty list for "Unit".
  (cond ((assoc exchange-structure-name list-of-types)
	 => (lambda (definition)
	      (match-let
		  (((type-name type-kind tag . slot-properties) definition))
		slot-properties)))
	(else
	 '())))

(define (summarize-action action-name)
  ;; Returns a list of (action-name signature action-property
  ;; request-properties response-properties).  A signature is a pair
  ;; of request/response names.  It renames the response name from
  ;; "XXXXOutput" to "XXXResponse".
  (when tr? (format #t ";; summarize-action ~a~%" action-name))
  (let* ((action-structure (find-action-structure action-name))
	 (properties1 (itemize-action-property action-structure))
	 (signature (make-exchange-signature action-structure))
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

;; This part makes a list LIST-OF-TYPES-IN-REQUESTS of type-names that
;; appear in requests.

;; LIST-ERROR-TYPES returns a list of error types defined.  Errors
;; have "traits": {"smithy.api#error": "client",
;; "smithy.api#httpError": 409}, where this is an entry for
;; "BucketAlreadyExists".

(define (collect-types-in-slots slot-properties acc)
  (if (null? slot-properties)
      acc
      (match-let (((slot tag type . _) (car slot-properties)))
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
		   ((type-name type-kind tag . slot-properties) definition))
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
	       (error "BAD type-kind" type-kind))))))))

(define (collect-types-in-requests request-properties acc)
  (if (null? request-properties)
      acc
      (collect-types-in-requests
       (cdr request-properties)
       (match-let* (((slot tag type . _) (car request-properties)))
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

(define (list-error-types)
  (delete-duplicates
   (apply-append
    (map (lambda (definition)
	   (match-let
	       (((type-name type-kind tag . slot-properties) definition))
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
;;; PARAMETER COLLECTOR
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
	   (error "get-uri-method-path: unknown url pattern found: ~s" uri)))))

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
	  (match-let (((slot tag type locus required) (car props)))
	    ;; (when tr? (format #t ";; slot=~s tag=~s locus=~s required=~s~%"
	    ;; slot tag locus required))
	    (if (not required)
		(loop (cdr props) queries-acc headers-acc)
		(case locus
		  ((PATH)
		   (loop (cdr props) queries-acc headers-acc))
		  ((QUERY)
		   (loop (cdr props)
			 (append queries-acc (list tag))
			 headers-acc))
		  ((HEADER)
		   (loop (cdr props)
			 queries-acc
			 (append headers-acc (list tag))))
		  ((HEADER-PREFIX)
		   (format #t "BAD httpPrefixHeaders marked required~%")
		   (error "BAD httpPrefixHeaders marked required"))
		  ((PAYLOAD)
		   (loop (cdr props) queries-acc headers-acc))
		  ((ELEMENT)
		   (loop (cdr props) queries-acc headers-acc))
		  (else
		   (format #t "BAD properties=~s~%" (car props))
		   (error "BAD properties" (car props))))))))))

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

;; MEMO about Golang's net/http server.  Headers can be accessed in
;; Request.Header which is type Header (a map).  Queries can be
;; accessed in Request.URL.Query() which is type Values (a map).

(define (make-variable-name s)
  (string-map (lambda (c) (if (or (eqv? c #\-) (eqv? c #\=)) #\_ c))
	      (string-downcase s)))

(define (make-fetch-condition source s)
  ;; Makes a condition on queries and headers.  A source is a variable
  ;; name holding queries "q" or headers "h".
  (assert (or (string=? source "q") (string=? source "h")))
  (cond ((string-contains s "=")
	 => (lambda (i)
	      (let* ((key (substring s 0 i))
		     (var (make-variable-name s))
		     (val (substring s (+ i 1))))
		(format #f "var ~a = (~a.Get(~s) == ~s)" var source key val))))
	(else
	 (let* ((key s)
		(var (make-variable-name key)))
	   (cond ((string=? source "q")
		  (format #f "var ~a = q.Has(~s)" var key))
		 ((string=? source "h")
		  (format #f "var ~a = (len(h.Values(~s)) != 0)" var key))
		 (else
		  (error "never")))))))

(define (make-conditionals queries-headers)
  ;;(format #t "make-conditionals ~s~%" queries-headers)
  (if (null? queries-headers)
      "true"
      (let ((v (map make-variable-name queries-headers)))
	;; (string-append "(" (string-join v " && ") ")")
	(string-append (string-join v " && ")))))

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
  ;; Generates a dispatcher for one pattern.  It returns a list of
  ;; lines of code.  The root path "/" is fixed to ServeMux's "/{$}".
  (match-let* ((((method path-raw) . dispatches) dispatch-entry)
	       (path (if (string=? path-raw "/") "/{$}" path-raw)))
    (let-values (((queries headers) (list-queries-headers dispatches)))
      (append
       ;; Handler registering code:
       (list
	(string-append
	 (format #f "sx.HandleFunc(\"~a ~a\"," method path)
	 " func(w http.ResponseWriter, r *http.Request) {"))
       ;; Check code for root-condition (NEVER GENERATED):
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
   (list (format #f "// dispatcher.go (~a)" generation-date)
	 "// API-STUB.  A dispatcher for net/http.ServeMux.  It"
	 "// switches handlers with regard to method-path patterns"
	 "// and parameters marked as required in API."
	 ;; A blank line is added to break from a package comment.
	 "")
   (delete
    ""
    (list (format #f "package ~a" bb-dispatcher-package)
	  "import ("
	  ;; "\"context\""
	  "\"net/http\""
	  (if (not (string=? bb-server-package bb-dispatcher-package))
	      (format #f "\"~a/~a\"" bb-package-path bb-server-package)
	      "")
	  ")"
	  "// REGISTER_DISPATCHER registers handers of BB-server to ServeMux."
	  (string-append
	   "func register_dispatcher"
	   (format #f "(bbs *~a, sx *http.ServeMux)" bb-server-type)
	   " error {")))
   (apply-append
    (map generate-dispatcher-entry list-of-dispatches))
   (list (format #f "return nil}"))))

(define (write-dispatcher port)
  (let ((ss (generate-dispatcher list-of-dispatches)))
    (format port "~a~%" (string-join ss "\n"))))

(define (display-dispatcher)
  (write-dispatcher #t))

(define (dump-dispatcher file)
  (call-with-output-file file
    (lambda (port)
      (write-dispatcher port))))

;;;
;;; HANDLER PRINTER
;;;

;; This part prints handler functions, which are called in the
;; dispatcher.

;; Handling input records is straightforward.  But, there are four
;; cases of handling a payload in a response.
;;
;; (1) "Unit" output type: Output type of Unit means a returned
;; response contains nothing.  EXAMPLE: "DeleteBucket".
;;
;; (2) A payload slot: A slot of an output record can be marked by
;; "smithy.api#httpPayload" in its traits.  It indicates the slot is
;; returned in a payload.  EXAMPLE: "CopyObject".  A record
;; "CopyObjectOutput" has "CopyObjectResult" slot.
;;
;; (3) A payload output record: An output record can be marked by
;; "smithy.api#xmlName" in its traits.  It indicates an output record
;; itself is returned in a payload, but its xml-tag is replaced by a
;; name cited.  EXAMPLE: "ListBuckets".  "ListBucketsOutput" is marked
;; with "ListAllMyBucketsResult".
;;
;; (4) Others: Other output records have no payload.  EXAMPLE:
;; "HeadObject".

(define (split-payload-properties properties)
  ;; Separates exchanges in a payload from exchanges in parameters
  ;; (headers/queries).
  (let-values
      (((payloads parameters)
	(partition (lambda (property)
		     (match-let (((slot tag type locus required) property))
		       (eqv? locus 'PAYLOAD)))
		   properties)))
    (assert (<= (length payloads) 1))
    (list (if (= (length payloads) 0) #f (car payloads))
	  parameters)))

(define (resolve-type type-name)
  ;; Returns a type representation.  Primitive types resolve to a type
  ;; name in Golang.  Composite types resolve to themselves that
  ;; should be defined types.
  (match-let* ((definition (assoc type-name list-of-types))
	       ((type-name type-kind tag . slot-properties) definition))
    (if (primitive-type? type-kind)
	type-kind
	(format #f "types.~a" type-name))))

(define (make-sdk-enumerator type-name enumerator-string)
  ;; Makes an enumerator of an enum of AWS-SDK.
  (format #f "types.~a~a" type-name (camelcase-string enumerator-string)))

(define (make-enumerator-intern-function type slot-properties)
  ;; Makes a function with clauses mapping a string to an enumerator.
  ;; A slot property of an enumeration-type is (slot tag "Unit" ...),
  ;; and a tag is a string representation of an enumerator.
  (append
   (list (format #f "func intern_~a(s string) (types.~a, error) {" type type)
	 (format #f "switch s {"))
   (map (lambda (property)
	  (match-let (((slot tag _ . _) property))
	    (let ((enumerator (make-sdk-enumerator type tag)))
	      (format #f "case ~s: return ~a, nil" tag enumerator))))
	slot-properties)
   (list
    (string-append
     (format #f "default: var err3 = &Bb_enum_intern_error")
     (format #f "{\"types.~a\", s}" type))
    "return \"_invalid_\", err3}}")))

(define (make-enumerator-interns list-of-types-in-requests)
  ;; Makes interning functions for enumerators.  Functions are named
  ;; with an enumeration name prefixed by "intern_".
  (apply-append
   (map (lambda (type)
	  (match-let*
	      ((definition (assoc type list-of-types))
	       ((type-name type-kind tag . slot-properties) definition))
	    (if (string=? type-kind "enum")
		(make-enumerator-intern-function type slot-properties)
		'())))
	list-of-types-in-requests)))

(define (make-coercing-intern name type-name assigner rhs)
  ;; Makes a coercion of a string to a given type.  Calling an
  ;; assigner makes an assignment.  It assumes a type-name is defined.
  (match-let* ((definition (assoc type-name list-of-types))
	       ((type-name type-kind tag . slot-properties) definition))
    (let ((error-record-clause
	   (string-append
	    "if err2 != nil {"
	    (format #f "input_errors[~s] = err2" name)
	    "}")))
      (cond
       ;; Primitive-types:
       ((string=? type-kind "blob")
	(error "make-coercing-intern with blob"))
       ((string=? type-kind "boolean")
	(list
	 (format #f "var s = ~a" rhs)
	 (format #f "var x, err2 = strconv.ParseBool(s)")
	 (string-append
	  error-record-clause
	  " else {"
	  (assigner "&x")
	  "}")))
       ((string=? type-kind "integer")
	(list
	 (format #f "var s = ~a" rhs)
	 (format #f "var x1, err2 = strconv.ParseInt(s, 10, 32)")
	 "var x2 = int32(x1)"
	 (string-append
	  error-record-clause
	  " else {"
	  (assigner "&x2")
	  "}")))
       ((string=? type-kind "long")
	(list
	 (format #f "var s = ~a" rhs)
	 (format #f "var x, err2 = strconv.ParseInt(s, 10, 64)")
	 (string-append
	  error-record-clause
	  " else {"
	  (assigner "&x")
	  "}")))
       ((string=? type-kind "operation")
	(error "make-coercing-intern: called with operation"))
       ((string=? type-kind "service")
	(error "make-coercing-intern: called with service"))
       ((string=? type-kind "string")
	(list
	 (assigner (format #f "h_thing_pointer(~a)" rhs))))
       ((string=? type-kind "timestamp")
	(list
	 (format #f "var s = ~a" rhs)
	 (format #f "var x, err2 = time.Parse(time.RFC3339, s)")
	 (string-append
	  error-record-clause
	  " else {"
	  (assigner "&x")
	  "}")))
       ;; Composite-types:
       ((string=? type-kind "enum")
	(list
	 (format #f "var s = ~a" rhs)
	 (format #f "var x, err2 = intern_~a(s)" type-name)
	 (string-append
	  error-record-clause
	  " else {"
	  (assigner "x")
	  "}")))
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
	(error "make-coercing-intern: called with unknown"))))))

(define (make-coercing-extern type-name assigner rhs)
  ;; Makes a coercion of a given type to a string.  It is in the
  ;; opposite direction of make-extern-coercing.  It errs when a
  ;; type-name is not defined.
  (match-let* ((definition (assoc type-name list-of-types))
	       ((type-name type-kind tag . slot-properties) definition))
    (cond
     ;; Primitive-types:
     ((string=? type-kind "blob")
      (error "make-coercing-extern with blob"))
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
      (error "make-coercing-extern with operation"))
     ((string=? type-kind "service")
      (error "make-coercing-extern with service"))
     ((string=? type-kind "string")
      (list
       (assigner (format #f "string(*~a)" rhs))))
     ((string=? type-kind "timestamp")
      (list
       (assigner (format #f "~a.Format(time.RFC3339)" rhs))))
     ;; Composite-types:
     ((string=? type-kind "enum")
      (list
       (assigner (format #f "string(~a)" rhs))))
     ((string=? type-kind "list")
      ;; Lists are never used.
      (error "make-coercing-extern with list")
      (list
       (assigner (format #f "~a" rhs))))
     ((string=? type-kind "map")
      ;; Maps are handled by a caller.
      (list
       (assigner (format #f "~a" rhs))))
     ((string=? type-kind "structure")
      (list
       (assigner (format #f "~a" rhs))))
     ((string=? type-kind "union")
      (list
       (assigner (format #f "~a" rhs))))
     (else
      (format #t "BAD type-kind ~s~%" type-kind)
      (error "make-coercing-extern with unknown")))))

(define (make-input-import request-property)
  ;; Makes an assignment in a structure "s3.XXXXInput" of AWS-SDK.  A
  ;; slot-property is a list of five-tuples (slot tag type locus
  ;; required).  Note the type name of a request is "XXXXRequest" in
  ;; the API and Smithy.
  (match-let* (((slot tag type locus required) request-property)
	       (definition (assoc type list-of-types))
	       ((type-name type-kind _ . slot-properties) definition)
	       (slot-name (if (not (eqv? tag #f)) tag slot)))
    (case locus
      ((PATH)
       ;; Path parameters are taken by request.PathValue(key).
       (let ((key-name
	      (cond ((string=? slot-name "Bucket") (string-downcase slot-name))
		    ((string=? slot-name "Key") (string-downcase slot-name))
		    (else
		     (error "make-input-import: path unknown" slot-name)))))
	 (list (format #f "{var x = r.PathValue(~s)" key-name)
	       (format #f "if x != \"\" {i.~a = &x}}" slot))))
      ((QUERY)
       (let ((rhs (format #f "qi.Get(~s)" slot-name))
	     (assigner (lambda (rhs)
			 (format #f "i.~a = ~a" slot rhs))))
	 (append
	  (list (format #f "if qi.Has(~s) {" slot-name))
	  (string-append-on-tail
	   (make-coercing-intern slot-name type assigner rhs) "}"))))
      ((HEADER)
       ;; (format #f "i.~a = hi.Get(~s)" slot slot-name)
       (cond
	((string=? type-kind "list")
	 ;; List's slot-properties is (("member" "member" type2 . _))
	 (match-let (((_ _ type2 . _) (car slot-properties)))
	   (let* ((element-type (resolve-type type2))
		  (assigner (lambda (rhs)
			      (format #f "bin = append(bin, ~a)" rhs))))
	     (append
	      (list (format #f "if len(hi.Values(~s)) != 0 {" slot-name)
		    (format #f "var rhs = hi.Values(~s)" slot-name)
		    (format #f "var bin []~a" element-type)
		    (format #f "for _, v := range slices.All(rhs) {"))
	      (string-append-on-tail
	       (make-coercing-intern slot-name type2 assigner "v") "}")
	      ;;(format #f "bin = append(bin, v)")
	      (list
	       (format #f "i.~a = bin}" slot))))))
	((string=? type-kind "map")
	 ;; NEVER THIS CASE.  MAPS ARE USED IN HEADER-PREFIX.
	 (error "make-input-import with map in headers"))
	(else
	 (let ((rhs (format #f "hi.Get(~s)" slot-name))
	       (assigner (lambda (rhs)
			   (format #f "i.~a = ~a" slot rhs))))
	   (append
	    (list (format #f "if len(hi.Values(~s)) != 0 {" slot-name))
	    (string-append-on-tail
	     (make-coercing-intern slot-name type assigner rhs) "}"))))))
      ((HEADER-PREFIX)
       ;; IT ASSUMES MAPS ARE ALWAYS OF STRINGS.
       (assert (string=? type-kind "map"))
       (match-let ((((_ _ type2 . _) (_ _ type3 . _)) slot-properties))
	 (let* ((key-type (resolve-type type2))
		(value-type (resolve-type type3))
		(_ (assert (string=? key-type "string")))
		(_ (assert (string=? value-type "string")))
		(map-type (format #f "map[~a]~a" key-type value-type))
		(assigner (lambda (rhs)
			    (format #f "bin[key] = ~a" rhs))))
	   (list (format #f "if len(hi.Values(~s)) != 0 {" slot-name)
		 (format #f "var prefix = http.CanonicalHeaderKey(~s)"
			 slot-name)
		 (format #f "var bin ~a" map-type)
		 (format #f "for k, v := range hi {")
		 (format #f "if strings.HasPrefix(k, prefix) {bin[k] = v[0]}}")
		 (format #f "i.~a = bin}" slot)))))
      ((PAYLOAD)
       (cond
	((string=? type-kind "blob")
	 (list (format #f "{i.~a = r.Body}" slot)))
	(else
	 ;; Records for a payload slot are: {CompletedMultipartUpload,
	 ;; CreateBucketConfiguration, Delete, Tagging}.
	 (list
	  ;; xml.Unmarshal() = xml.NewDecoder().Decode().
	  (format #f "{var x types.~a" type)
	  "var err1 = xml.NewDecoder(r.Body).Decode(&x)"
	   "if err1 != nil {"
	   (string-append
	    "if err1 != io.EOF {"
	    (format #f "input_errors[~s] = fmt.Errorf" "_payload_")
	    (format #f "(\"Malformed http body for types.~a: %w\", err1)}"
		    type))
	   (format #f "} else {i.~a = &x}}" slot)))))
      ((ELEMENT)
       (error "make-input-import; bad locus ELEMENT" request-property))
      (else
       (error "make-input-import; bad locus ELEMENT" request-property)))))

(define (make-output-export response-property)
  ;; Makes extraction code from structure "s3.XXXXOutput" of AWS-SDK.
  ;; Each property is a list of five-tuples (slot tag type locus
  ;; required).  Note "XXXXOutput" is copied and stored in variable
  ;; "s".
  (match-let* (((slot tag type locus required) response-property)
	       (definition (assoc type list-of-types))
	       ((type-name type-kind _ . slot-properties) definition))
    ;;(when tr? (format #t ";; . make-output-export ~s~%" slot))
    (case locus
      ((PATH)
       (error "make-output-export: path in response" tag))
      ((QUERY)
       (error "make-output-export; bad query in response" tag))
      ((HEADER)
       ;;(list (format #f "ho.Add(~s, o.~a)" tag slot))
       (cond
	((string=? type-kind "enum")
	 (let ((rhs (format #f "o.~a" slot))
	       (assigner (lambda (rhs)
			   (format #f "ho.Add(~s, ~a)" tag rhs))))
	   (append
	    (list (format #f "if ~a != \"\" {" rhs))
	    (string-append-on-tail
	     (make-coercing-extern type assigner rhs) "}"))))
	((string=? type-kind "list")
	 (let ((rhs (format #f "o.~a" slot))
	       (assigner (lambda (rhs)
			   (format #f "ho.Add(~s, ~a)" tag rhs))))
	   (append
	    (list (format #f "if len(~a) != 0 {" rhs))
	    (string-append-on-tail
	     (make-coercing-extern type assigner rhs) "}"))))
	((string=? type-kind "map")
	 ;; NEVER THIS CASE.  MAPS ARE USED IN HEADER-PREFIX.
	 (error "make-output-export: map for header"))
	(else
	 (let ((rhs (format #f "o.~a" slot))
	       (assigner (lambda (rhs)
			   (format #f "ho.Add(~s, ~a)" tag rhs))))
	   (append
	    (list (format #f "if ~a != nil {" rhs))
	    (string-append-on-tail
	     (make-coercing-extern type assigner rhs) "}"))))))
      ((HEADER-PREFIX)
       (assert (string=? type-kind "map"))
       (list (format #f "for k, v := range o.~a {" slot)
	     (format #f "ho.Add(k, v)}")))
      ((PAYLOAD)
       (begin
	 (when tr? (format #t ";; Payload in response: ~s~%" tag))
	 '()))
      ((ELEMENT)
       (begin
	 ;; (when tr? (format #t ";; Skip element in response: ~s~%" tag))
	 '()))
      (else
       (error "make-output-export: bad properties" response-property)))))

(define (make-unit-response code)
  ;; Makes an empty response for an output of "Unit" type.
  (list (format #f "var status int = ~a" code)
	"w.WriteHeader(status)"))

(define (make-payload-response http-status response-type slot-property)
  ;; Makes a response of a payload.  Payload is an output record
  ;; itself or a designated slot.  It uses an output record when
  ;; slot-property=#f.
  (cond
   ((eqv? #f slot-property)
    ;; (1) Payload is an output record (including unit case).
    (match-let* ((output-type (adjust-output-structure-name response-type))
		 (definition (assoc output-type list-of-types))
		 ((type-name type-kind tag . slot-properties) definition))
      (cond
       ((eqv? #f tag)
	(format #t ";; No payload response: ~s~%" output-type)
	(list (format #f "var status int = ~a" http-status)
	      "w.WriteHeader(status)"))
       (else
	;; Marshaling errors are implementation errors.
	(list "ho.Set(\"Content-Type\", \"application/xml\")"
	      (format #f "var s = h_~a(*o)" response-type)
	      (string-append
	       "var ox, err6 = xml.MarshalIndent"
	       (format #f "(s, \" \", \"  \")"))
	      "if err6 != nil {log.Fatal(err6)}"
	      (format #f "var status int = ~a" http-status)
	      "w.WriteHeader(status)"
	      "var _, err7 = w.Write([]byte(xml.Header))"
	      "if err7 != nil {bbs.cope_write_error(ctx, w, r, err7)}"
	      "var _, err8 = w.Write(ox)"
	      "if err8 != nil {bbs.cope_write_error(ctx, w, r, err8)}")))))
   (else
    ;; (2) Payload is described by a slot-property.
    (match-let* (((slot tag type locus required) slot-property)
		 (return-binary? (string=? type "StreamingBlob")))
      (cond
       (return-binary?
	(list "ho.Set(\"Content-Type\", \"application/octet-stream\")"
	      (format #f "var status int = ~a" http-status)
	      "w.WriteHeader(status)"
	      (format #f "var _, err7 = io.Copy(w, o.~a)" slot)
	      "if err7 != nil {bbs.cope_write_error(ctx, w, r, err7)}"))
       (else
	;; Marshaling errors means implementation errors.
	(list "ho.Set(\"Content-Type\", \"application/xml\")"
	      (string-append
	       "var ox, err6 = xml.MarshalIndent"
	       (format #f "(o.~a, \" \", \"  \")" slot))
	      "if err6 != nil {log.Fatal(err6)}"
	      (format #f "var status int = ~a" http-status)
	      "w.WriteHeader(status)"
	      "var _, err7 = w.Write([]byte(xml.Header))"
	      "if err7 != nil {bbs.cope_write_error(ctx, w, r, err7)}"
	      "var _, err8 = w.Write(ox)"
	      "if err8 != nil {bbs.cope_write_error(ctx, w, r, err8)}")))))))

(define (make-handler-function action)
  (match-let*
      (((name signature action-property request-properties response-properties)
	action)
       ((request-type response-type) signature)
       (input-type (adjust-input-structure-name request-type))
       (output-type (adjust-output-structure-name response-type))
       ((payload-import-property import-properties)
	(split-payload-properties request-properties))
       ((payload-export-property export-properties)
	(split-payload-properties response-properties))
       ((_ _ http-status) action-property))
    (when tr? (format #t ";; make-handler-function ~s~%" name))
    (append
     ;; Start of function declaration:
     (list (string-append
	    (format #f "func h_~a" name)
	    (format #f "(bbs *~a, w http.ResponseWriter, r *http.Request) {"
		    bb-server-type))
	   "var qi = r.URL.Query()"
	   "var hi = r.Header"
	   "var ho = w.Header()"
	   "// Mark variables used to avoid unused errors:"
	   "var _, _, _ = qi, hi, ho"
	   "var input_errors = map[string]error{}"
	   "var ctx1 = r.Context()"
	   (string-append
	    "var ctx2 = context.WithValue(ctx1, \"request-id\","
	    " bbs.make_request_id())")
	   (string-append
	    "var ctx = context.WithValue(ctx2, \"input-errors\","
	    " input_errors)"))
     ;; Input accessors:
     (list (format #f "var i = s3.~a{}" input-type))
     (apply-append (map make-input-import import-properties))
     (if payload-import-property
	 (make-input-import payload-import-property)
	 '())
     (list "if len(input_errors) > 0 {"
	   "bbs.respond_on_input_error(ctx, w, r, input_errors)"
	   "return}")
     ;; Hander invocation:
     (list (if (string=? output-type "Unit")
	       (format #f "var _, err5 = bbs.~a(ctx, &i)" name)
	       (format #f "var o, err5 = bbs.~a(ctx, &i)" name))
	   "if err5 != nil {"
	   "bbs.respond_on_action_error(ctx, w, r, err5)"
	   "return}")
     ;; Output accessors:
     (cond
      ((string=? output-type "Unit")
       ;; Note DeleteBucket has "Unit" output.
       (assert (eqv? #f payload-export-property))
       (make-unit-response http-status))
      (else
       (append
	(apply-append (map make-output-export export-properties))
	(make-payload-response http-status response-type
			       payload-export-property))))
     ;; Function end:
     (list "}"))))

(define (make-auxiliary-functions)
  (append
   (if #f
       (list
	"// GATHER_HEADERS gathers entries of headers that match a prefix."
	"func gather_headers(h http.Header, prefix string) map[string]string {"
	"var p = http.CanonicalHeaderKey(prefix)"
	"var m map[string]string"
	"for k, v := range h {"
	"if strings.HasPrefix(k, p) {"
	"m[k] = v[0]}}"
	"return m}")
       '())))

(define (make-handler-file list-of-actions)
  (append
   (list (format #f "// handler.go (~a)" generation-date)
	 "// API-STUB.  Handler functions (h_XXXX) called from the"
	 "// dispatcher."
	 ;; A blank line is added to break from a package comment.
	 "")
   (delete
    ""
    (list (format #f "package ~a" bb-dispatcher-package)
	  "import ("
	  "\"context\""
	  "\"encoding/xml\""
	  "\"fmt\""
	  "\"io\""
	  "\"log\""
	  "\"net/http\""
	  "\"slices\""
	  "\"strconv\""
	  "\"strings\""
	  "\"time\""
	  (if (not (string=? bb-server-package bb-dispatcher-package))
	      (format #f "\"~a/~a\"" bb-package-path bb-server-package)
	      "")
	  "\"github.com/aws/aws-sdk-go-v2/service/s3\""
	  "\"github.com/aws/aws-sdk-go-v2/service/s3/types\""
	  ")"))
   (list "// BB_ENUM_INTERN_ERROR is an error returned when interning"
	 "// an enumeration."
	 "type Bb_enum_intern_error struct {"
	 "Enum string"
	 "Name string"
	 "}"
	 "func (e *Bb_enum_intern_error) Error() string {"
	 (string-append
	  "return \"Enum \" + e.Enum + \" unknown key: \""
	  " + strconv.Quote(e.Name)}"))
   #|(list "// BB_INPUT_ERROR is recorded in a context when an error"
	 "// occurs on interning a parameter."
	 "type Bb_input_error struct {"
	 "Key string"
	 "Err error"
	 "}"
	 "func (e *Bb_input_error) Error() string {"
	 "return \"Parameter \" + e.Key + \" error: \" + e.Err.Error()}")|#
   ;;"// RECORD_INPUT_ERROR is called on an error on interning a"
   ;;"// parameter to record it in the context."
   ;;(string-append
   ;; "func h_record_input_error"
   ;; "(ctx context.Context, key string, e error) {")
   ;;"var v = ctx.Value("input-errors").(*[]error)"
   ;;"*v = append(*v, Bb_input_error{key, e})}"
   ;;"var m = ctx.Value(\"input-errors\").(map[string]error)"
   ;;"m[key] = &Bb_input_error{key, e}}"
   (apply-append
    (map make-handler-function list-of-actions))))

(define (write-handlers port)
  (let ((ss (make-handler-file list-of-actions)))
    (format port "~a~%" (apply string-append (intervene-separator "\n" ss)))))

(define (write-enumerator-interns port)
  (let ((ss (make-enumerator-interns list-of-types-in-requests)))
    (format port "~a~%" (apply string-append (intervene-separator "\n" ss)))))

(define (write-auxiliary-functions port)
  (let ((ss (make-auxiliary-functions)))
    (format port "~a~%" (apply string-append (intervene-separator "\n" ss)))))

(define (dump-handlers file)
  (call-with-output-file file
    (lambda (port)
      (write-handlers port)
      (write-enumerator-interns port)
      (write-auxiliary-functions port))))

(define (display-handlers)
  (write-handlers #t))

(define (display-handler-function action)
  (let ((ss (make-handler-function action)))
    (format #t "~a~%" (apply string-append (intervene-separator "\n" ss)))))

;; (display-handler-function (assoc "ListParts" list-of-actions))

;;;
;;; RESPONSE MARSHALER PRINTER
;;;

(define (check-output-whole-payload response-properties)
  ;; Checks if a slot is a whole payload in a response.  "CopyObject"
  ;; has "CopyObjectResult", "GetObject" has "Body", and
  ;; "UploadPartCopy" has "CopyPartResult".
  (let ((check-output-whole-payload1
	 (lambda (property)
	   (match-let (((slot tag type locus required) property))
	     (eqv? locus 'PAYLOAD)))))
    (any check-output-whole-payload1 response-properties)))

(define (check-xml-tag-needs-correction definition)
  ;; Checks if the type has an element which needs correction of
  ;; XML-tag on an array.
  ;;
  ;; An example of type definition is "TagSet" in "Tagging" type.  Their
  ;; definitions are:
  ;; ("Tagging" "structure" #f ("TagSet" #f "TagSet" ELEMENT #t))
  ;; ("TagSet" "list" #f ("member" "Tag" "Tag" ELEMENT #f)).
  (match-let* (((type-name type-kind tag . slot-properties) definition))
    (cond
     ((or (primitive-type? type-kind)
	  (string=? type-kind "enum")
	  (string=? type-kind "union")
	  (string=? type-kind "map"))
      #f)
     ((string=? type-kind "structure")
      (any check-xml-tag-needs-correction-in-element slot-properties))
     ((string=? type-kind "list")
      (match-let ((((slot xml-tag type locus required) . _) slot-properties))
	(if (not (eqv? xml-tag #f))
	    #t
	    (any check-xml-tag-needs-correction-in-element slot-properties))))
     (else
      (error "BAD type-kind definition" definition)))))

(define (check-xml-tag-needs-correction-in-element property)
  (match-let* (((slot tag type locus required) property)
	       (definition (assoc type list-of-types)))
    (check-xml-tag-needs-correction definition)))

(define (make-slot-marshaler property)
  ;; Returns lines of marshaler for an response element.  (* FALSE
  ;; STATEMENT: It specially treats arrays (kind="list"), as
  ;; "EncodeElement" puts a tag on an array not by the type name but
  ;; by the passed start tag. *)
  (match-let* (((slot tag type locus required) property)
	       (definition (assoc type list-of-types))
	       ((type-name type-kind _ . slot-properties) definition)
	       (null-value (if (string=? type-kind "enum") "\"\"" "nil"))
	       (slot-name (if (not (eqv? tag #f)) tag slot))
	       (fix-tag (check-xml-tag-needs-correction definition)))
    ;;(format #t ";; make-slot-marshaler ~s ~s~%" property definition)
    (case locus
      ((PATH QUERY)
       (format #t "BAD property in response: ~s~%" property)
       (error "make-slot-marshaler: BAD property in response" property))
      ((PAYLOAD)
       (list
	(string-append
	 (format #f "{var err2 = e.EncodeElement")
	 (format #f "(s.~a, h_make_tag(\"~a\"))" slot slot-name))
	(format #f "if err2 != nil {return err2}}")))
      ((HEADER HEADER-PREFIX)
       '())
      ((ELEMENT)
       (cond
	(fix-tag
	 (format #t ";; TAG CORRECTION NEEDED: ~s~%" definition)
	 (list
	  (format #f "if s.~a != ~a {" slot null-value)
	  (format #f "var err2 = export_~a(e, s.~a)" type-name slot)
	  "if err2 != nil {return err2}}"))
	;; ((string=? type-kind "list")
	;;  (list
	;;   (format #f "if s.~a != ~a {" slot null-value)
	;;   (format #f "var tag2 = h_make_tag(\"~a\")" slot-name)
	;;   "var err2 = e.EncodeToken(tag2)"
	;;   "if err2 != nil {return err2}"
	;;   (format #f "var err3 = e.Encode(s.~a)" slot)
	;;   "if err3 != nil {return err3}"
	;;   "var err4 = e.EncodeToken(tag2.End())"
	;;   "if err4 != nil {return err4}}"))
	(else
	 (list
	  (format #f "if s.~a != ~a {" slot null-value)
	  (string-append
	   (format #f "var err2 = e.EncodeElement")
	   (format #f "(s.~a, h_make_tag(\"~a\"))" slot slot-name))
	  (format #f "if err2 != nil {return err2}}")))))
      (else
       (format #t "BAD property in response: ~s~%" property)
       (error "make-slot-marshaler: BAD property in response" property)))))

(define (make-marshaler-function action)
  ;; Returns lines of a response marshaler for "XXXXResponse".
  (match-let*
      (((name (request-type response-type) _ _ response-properties) action)
       (output-type (adjust-output-structure-name response-type))
       (encoders (delete '() (map make-slot-marshaler response-properties)))
       (whole-payload (check-output-whole-payload response-properties)))
    (assert (or (not whole-payload) (= (length encoders) 1)))
    (cond
     ((string=? output-type "Unit")
      '())
     (else
      (match-let*
	  ((definition (assoc output-type list-of-types))
	   ((type-name1 type-kind tag . slot-properties) definition))
	(cond
	 ((and (= (length encoders) 0) (eqv? #f tag))
	  (format #t ";; Skip making marshaler: ~s~%" name)
	  '())
	 (else
	  (append
	   (list
	    (format #f "type h_~a s3.~a" response-type output-type)
	    (string-append
	     (format #f "func (s h_~a) MarshalXML" response-type)
	     (format #f "(e *xml.Encoder, _ xml.StartElement) error {")))
	   (cond
	    ((not (eqv? #f tag))
	     (append
	      (list (format #f "var tag1 = h_make_tag(~s)" tag)
		    "var err1 = e.EncodeToken(tag1)"
		    "if err1 != nil {return err1}")
	      (apply-append encoders)
	      (list "var err9 = e.EncodeToken(tag1.End())"
		    "if err9 != nil {return err9}")))
	    ((not (null? encoders))
	     (apply-append encoders)))
	   (list
	    (format #f "return nil}"))))))))))

(define (make-marshaler-file list-of-actions)
  (append
   (list (format #f "// marshaler.go (~a)" generation-date)
	 "// API-STUB.  Marshalers of response structures.  Response"
	 "// structures need custom marshalers, because they have"
	 "// some slots that need to be renamed and also have an"
	 "// extra slot that should be suppressed."
	 ;; A blank line is added to break from a package comment.
	 "")
   (list (format #f "package ~a" bb-dispatcher-package)
	 "import ("
	 ;; "\"context\""
	 "\"encoding/xml\""
	 "\"github.com/aws/aws-sdk-go-v2/service/s3\""
	 ")"
	 "func h_thing_pointer[T any](v T) *T {return &v}"
	 "func h_make_tag(k string) xml.StartElement {"
	 "return xml.StartElement{Name: xml.Name{Local: k}}}")
   (apply-append
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
;; (make-marshaler-function (assoc "ListParts" list-of-actions))
;; (display-marshalers)

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
	(format #f "func (bbs *~a) ~a" bb-server-name name)
	(format #f "(ctx context.Context, params *s3.~a," input-name)
	(format #f " optFns ...func(*s3.Options))")
	(format #f " (*s3.~a, error) {" api-output-name))
       (format #f "var o = s3.~a{}" api-output-name)
       "return &o, nil}"))))

(define (make-api-template-file list-of-actions)
  (append
   (list (format #f "// api-template.go (~a)" generation-date)
	 "// API-STUB.  Handler templates. They should be replaced by"
	 "// actual implementations."
	 ;; A blank line is added to break from a package comment.
	 "")
   (delete
    ""
    (list (format #f "package ~a" bb-server-package)
	  "import ("
	  "\"context\""
	  "\"github.com/aws/aws-sdk-go-v2/service/s3\""
	  ")"))
   (list "type Bb_server struct {}"
	 "// MAKE_REQUEST_ID makes a new request-id."
	 (string-append
	  "func (bbs *Bb_server) make_request_id() string {"
	  "panic(e)}")
	 "// RESPOND_ON_ACTION_ERROR is called on an action error and"
	 "// makes a response for it."
	 (string-append
	  "func (bbs *Bb_server) respond_on_action_error"
	  "(ctx context.Context, w http.ResponseWriter,"
	  " r *http.Request, e error) {"
	  "panic(e)}")
	 "// RESPOND_ON_INPUT_ERROR is called on an input error and"
	 "// makes a response for it."
	 (string-append
	  "func (bbs *Bb_server) respond_on_input_error"
	  "(ctx context.Context, w http.ResponseWriter,"
	  " r *http.Request, m map[string]error) {"
	  "panic(m)}")
	 "// COPE_WRITE_ERROR is called on a write error of response"
	 "// payload and makes a response for it."
	 (string-append
	  "func (bbs *Bb_server) cope_write_error"
	  "(ctx context.Context, w http.ResponseWriter,"
	  " r *http.Request, e error) {"
	  "panic(e)}"))
   ;;"// RESPOND_ON_INPUT_ERROR is called on an error on"
   ;;"// interning enumerations and makes a response for it."
   ;;(string-append
   ;;"func (bbs *Bb_server) respond_on_input_error"
   ;;"(ctx context.Context, w http.ResponseWriter,"
   ;;" r *http.Request, name string) {"
   ;;"panic(fmt.Errorf(\"Bad parameter %s\", name))}")
   ;;"// RESPOND_ON_MISSING_INPUT is called on an internal error"
   ;;"// and makes a response for it."
   ;;(string-append
   ;;"func (bbs *Bb_server) respond_on_missing_input"
   ;;"(ctx context.Context, w http.ResponseWriter,"
   ;;" r *http.Request, name string) {"
   ;;"panic(fmt.Errorf(\"Missing path: %s\", name))}")
   ;;"// RECORD_INPUT_ERROR is called on an error on interning a"
   ;;"// parameter to record it in the context."
   ;;"var v = ctx.Value("input-errors").(*[]Bb_input_error_record)"
   ;;"*v = append(*v, Bb_input_error_record{key, e})}"
   ;;(string-append
   ;;	  "func record_input_error"
   ;;	  "(ctx context.Context, key string, e error) {"
   ;;	  "var m = ctx.Value(\"input-errors\").(map[string]error)"
   ;;	  "v[key] = e}"))
   (apply-append
    (map make-api-template list-of-actions))))

(define (write-api-template-file port list-of-actions)
  (let ((ss (make-api-template-file list-of-actions)))
    (format port "~a~%" (apply string-append (intervene-separator "\n" ss)))))

(define (dump-template file)
  (call-with-output-file file
    (lambda (port)
      (write-api-template-file port list-of-actions))))

;;;
;;; WHOLE TASK
;;;

;; (define list-of-types (make-type-definition-list s3-api))
;; (define list-of-actions (map summarize-action list-of-action-names))
;; (define list-of-types-in-requests (list-types-in-requests))
;; (define collected-dispatches (collect-request-dispatches list-of-actions))
;; (define merged-dispatches (merge-request-dispatches list-of-actions))
;; (define list-of-dispatches (sort-dispatches merged-dispatches))

(define (dump-stub)
  (dump-dispatcher "dispatcher.go")
  (dump-handlers "handler.go")
  (dump-marshalers "marshaler.go")
  (dump-template "api-template.go"))
