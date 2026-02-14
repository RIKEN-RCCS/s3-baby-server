;; check.scm (2025-10-16)

;; Check runner.  It runs test cases described in json.

;; This is for "guile --r7rs", version 3.0.9 and later.  It uses
;; "spawn" which is introduced in guile-3.0.9.  We choose Guile just
;; because the base language (Scheme) is stable for years.

;; DESCRIPTION: This checker runs "aws" command with subcommands "s3"
;; or "s3api" as specified in the "operation" field, and expects an
;; output in "expect".  "expect" field is a simple pattern.
;; Mismatch in "expect" is an error.  "status" is an exit status, and
;; it is usually zero.
;;
;; "output" specifies an interpretation of "expect" field.  By "json",
;; "expect" is a record of regexp patterns in json.  By "lines",
;; "expect" is a vector of (non-json) lines of regexp patterns.
;; Patterns may include simple templates: "#|datetime|" matches
;; date-time, and "#_" is a wildcard, and so on.
;;
;; "record" field is used to remember values in the output.  It is a
;; list of pairs, with the first part a variable name and the second
;; part a path in json.  It remembers a value of the output at the
;; path in json in the named variable.
;;
;; Fields of "ID" and "step" are remarks.

;; MEMO: AWS CLI s3 command has "--quiet" option but it is too quiet.
;; And, "--only-show-errors", too.

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
 ;;(only (srfi srfi-43) vector-map) ;; vector-library
 ;;(srfi srfi-133) ;; r7rs-vector-library (NO srfi-133 in Guile)
 (only (rnrs base) infinite? assert)
 (srfi srfi-1) ;; list
 (srfi srfi-11) ;; multiple-values
 ;;(srfi srfi-28) ;; format
 (srfi srfi-60) ;; integers as bits
 )

;; Importing "(scheme base)" is (likely) not necessary when R7RS.

;; (import (system vm trace))
;; (import (system vm trap-state))
;; (add-trace-at-procedure-call! f)
;; (trace-calls-to-procedure match-to-template)

;; Make encoding ISO-8859-1.
;;(fluid-set! %default-port-encoding #f)

;;(setlocale LC_ALL "en_US.UTF-8")
(setlocale LC_ALL "C.utf-8")

(define (assume . bs) '())
(define (%read-error? x)
  (read-error? x))
(define (valid-number? string)
  (number? (string->number string)))
(load "srfi-180-body.scm")

(define (foldl f init list)
  ;; foldl : ('a * 'b -> 'b) -> 'b -> 'a list -> 'b
  (match list
    (() init)
    ((fst . rst) (foldl f (f fst init) rst))))

(define (foldr f init list)
  ;; foldr : ('a * 'b -> 'b) -> 'b -> 'a list -> 'b
  (match list
    (() init)
    ((fst . rst) (f fst (foldr f init rst)))))

(define (extend-alist k v alist)
  ;; It uses equal? as the comparator adhering to assoc.
  (acons k v (remove (lambda (pair) (equal? k (car pair))) alist)))

(define (append-string-vector v sep)
  ;; Appends strings in a vector with an intervening separator.
  (if (= (vector-length v) 0)
      ""
      (let* ((t (list (vector-ref v (- (vector-length v) 1))))
	     (v1 (vector->list v 0 (- (vector-length v) 1)))
	     (x (foldr (lambda (a b) (cons a (cons sep b))) t v1)))
	(apply string-append x))))

(define date-regexp "[0-9]{4}-[0-9]{2}-[0-9]{2}") ;; "2025-08-20"
(define time-regexp "[0-9]{2}:[0-9]{2}:[0-9]{2}") ;; "08:32:06"

(define datetime-regexp
  (string-append
   date-regexp
   "T"
   time-regexp
   "(\\.[0-9]{6})?"
   "\\+"
   "[0-9]{2}:[0-9]{2}"))

(define (run-system command)
  ;; Runs a command in a subprocess.  It returns three-values of
  ;; status and strings of stdout and stderr.  It assumes a command
  ;; finishes shortly, (it does not timeout).
  (let* ((shell "/bin/ksh"))
    (call-with-port (tmpfile)
      (lambda (outp)
	(call-with-port (tmpfile)
	  (lambda (errp)
	    (with-exception-handler
	     (lambda (x)
	       (display x) (newline))
	     (lambda ()
	       (let* ((pid (spawn shell (list shell "-c" command)
				  #:output outp
				  #:error errp))
		      (status (waitpid pid))
		      (_ (seek outp 0 SEEK_SET))
		      (_ (seek errp 0 SEEK_SET))
		      (stdout (get-string-all outp))
		      (stderr (get-string-all errp)))
		 (values (cdr status) stdout stderr))))))))))

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

(define (substitute-strings s keyvals)
  ;; Substitute strings specified by an alist.
  (if (null? keyvals)
      s
      (let* ((pair (car keyvals))
	     (key (car pair))
	     (val (cdr pair)))
	(substitute-strings (substitute-string s key val) (cdr keyvals)))))

(define (substitute-regexp-token s)
  ;; Replaces pattern tokens in a string with their regexp patterns.
  ;; It returns a pattern "^...$" for matching an entire line.  Tokens
  ;; are "#|datetime|", "#|date|", "#|time|", etc.  (TOKEN PATTERNS
  ;; SHOULD BE LISTED IN LONGER TO SHORTER ORDER).
  (let ((replacements `(("#|datetime|" . ,datetime-regexp)
			("#|date|" . ,date-regexp)
			("#|time|" . ,time-regexp)
			("#|etag|" . "(\")?[[:xdigit:]]{32}(\")?")
			("#|csum|" . "[a-zA-Z0-9+/=]{12}"))))
    (let ((s1 (substitute-strings s replacements)))
      (string-append "^" s1 "$"))))

(define (match-to-string expect result)
  ;; Matches a line of the result.  It returns a boolean.  It checks a
  ;; match is the whole string, in order to detect erroneous regexps.
  (cond ((string-match (substitute-regexp-token expect) result)
	 => (lambda (m)
	      (= (cdr (vector-ref m 1)) (string-length (vector-ref m 0)))))
	(else #f)))

(define match-to-template-tracing #f)

(define (match-to-template expect result)
  ;; (* Prints traces of match-to-template.  See match-to-template1. *)
  (let ((v (match-to-template1 expect result)))
    (when match-to-template-tracing
      (format #t "match-to-template expect=~s result=~s => ~s~%"
	      expect result v))
    v))

(define (match-to-template1 expect result)
  ;; Matches an expected template to a result, both in json.  It
  ;; returns a boolean.  A string pattern "#_" is a wildcard matching
  ;; any entity.  A string is a regexp pattern, which may include
  ;; pattern tokens like "#|time|".  A vector pattern should be empty
  ;; or singleton.  A vector pattern ["#_"] matches any vector
  ;; including empty ones.  A empty pattern "{}" matches an empty
  ;; result "" as well as an empty object "{}".  Note the json reader
  ;; returns #<eof> for an empty string.  An object pattern accepts
  ;; excess entities.
  (cond
   ((eqv? expect '())
    (or (eqv? result '()) (eof-object? result)))
   ((eqv? expect 'null)
    (eqv? expect result))
   ((boolean? expect)
    (eqv? expect result))
   ((number? expect)
    (eqv? expect result))
   ((string? expect)
    (cond ((string=? expect "#_")
	   #t)
	  ((string? result)
	   (match-to-string expect result))
	  (else #f)))
   ((list? expect)
    (cond ((list? result)
	   (let loop ((slots expect))
	     (if (null? slots)
		 #t
		 (match slots
		   (((key . expect1) . rest)
		    (cond ((assoc key result)
			   => (lambda (p)
				(let ((result1 (cdr p)))
				  (if (match-to-template expect1 result1)
				      (loop rest)
				      #f))))
			  (else #f)))
		   (else
		    (format #t "BAD template (not an alist): (~s)~%" slots)
		    #f)))))
	  (else #f)))
   ((vector? expect)
    (cond ((vector? result)
	   (cond ((= (vector-length expect) 0)
		  (= (vector-length result) 0))
		 ((= (vector-length expect) 1)
		  (let ((expect1 (vector-ref expect 0))
			(n (vector-length result)))
		    (cond ((and (string? expect1) (string=? expect1 "#_"))
			   #t)
			  ((= (vector-length result) 0)
			   #f)
			  (else
			   (let loop ((i 0))
			     (if (< i n)
				 (let ((result1 (vector-ref result i)))
				   (if (match-to-template expect1 result1)
				       (loop (+ i 1))
				       #f))
				 #t))))))
		 (else
		  (format #t "BAD template (long vector): (~s)~%" expect)
		  #f)))
	  (else #f)))
   (else
    (format #t "BAD template: expect=~s~%" expect)
    #f)))

(define (make-entity-value v)
  (cond ((eqv? v 'null) #f)
	((eqv? v #f) 'false)
	((eqv? v #t) 'true)
	(else v)))

(define (fetch-assoc entity slot)
  ;; Does assoc, but "null" value is treated as missing key.  It maps
  ;; #f to 'false and #t to 'true, to use false as a valid value.
  (cond ((assoc slot entity)
	 => (lambda (pair) (make-entity-value (cdr pair))))
	(else #f)))

(define (translate-expectation-slot e output-format)
  ;; Converts an expectation pattern in "expect" slot in an article
  ;; with regard to the output-format.  Output-format is either "json"
  ;; for a json object or "lines" for pattern strings.
  (cond ((string=? output-format "json")
	 e)
	((string=? output-format "lines")
	 (string-append (append-string-vector e "\n") "\n"))
	(else
	 (error "BAD: bad output-format slot" format))))

(define (fetch-json-slot entity path i)
  ;; Accesses for a path of keys in an entity.  A key is a string or a
  ;; number, and it access in a vector when a key is a number.  Note
  ;; the key part of an entity is a symbol.  It returns 'true or
  ;; 'false for boolean values.  Call this with i=0.
  (format #t "fetch-json-slot ~s ~s ~s~%" entity path i)
  (if (= i (vector-length path))
      entity
      (let ((key (vector-ref path i)))
	(cond ((number? key)
	       (vector-ref entity key))
	      ((fetch-assoc entity (string->symbol key))
	       => (lambda (entity1)
		    (fetch-json-slot entity1 path (+ i 1))))
	      (else
	       (format #t "BAD entity slot: path=~s~%" path)
	       #f)))))

(define (record-values env entity records i)
  ;; Extends key-value bindings by making an alist by fetching values
  ;; from the entity.  It returns an exteded bindings.  It checks an
  ;; entry one-by-one in records.  Call this with i=0.  An entry in
  ;; records is an alist and needs double parentheses in matching.
  (if (= i (vector-length records))
      env
      (match (vector-ref records 0)
	(((key . path))
	 (let ((var (string-append "#" (symbol->string key)))
	       (val (fetch-json-slot entity path 0)))
	   (format #t "recording key=~s value=~s~%" var val)
	   (record-values (extend-alist var val env) entity records (+ i 1))))
	(else
	 (format #t "BAD record slot: record=~s~%" records)
	 env))))

(define (check-run article i env test-loop)
  ;; Note (article = (vector-ref tests i)).
  (let* ((op
	  (cond ((assoc 'operation article)
		 => (lambda (pair) (substitute-strings (cdr pair) env)))
		(else (error "BAD: No operation slot" article))))
	 (output-format
	  (cond ((assoc 'output article)
		 => (lambda (pair) (cdr pair)))
		(else (error "BAD: No format slot" article))))
	 (expect
	  (cond ((fetch-assoc article 'expect)
		 => (lambda (e)
		      (translate-expectation-slot e output-format)))
		(else (error "BAD: No outcome slot" article))))
	 (records
	  (cond ((fetch-assoc article 'record)
		 => (lambda (e) e))
		(else #())))
	 (skip-flag (assoc 'skip article))
	 (stop-flag (assoc 'stop article)))
    (cond
     ((not (eqv? stop-flag #f))
      (format #t "STOP TESTING~%")
      #t)
     ((not (eqv? skip-flag #f))
      (format #t "Skipping test ~s~%" i)
      (test-loop (+ i 1) env))
     ((not (eqv? op #f))
      (format #t "testing: ~s ~s~%"
	      (assoc 'ID article)
	      (assoc 'step article))
      (format #t "environment: ~s~%" env)
      (format #t "expect: ~s~%" expect)
      (format #t "operation: ~s~%" op)
      (let-values (((status outs errs) (run-system op)))
	(format #t "status: ~s (~s, ~s)~%" status
		(status:exit-val status) (status:term-sig status))
	(format #t "stdout: ~s~%" outs)
	(format #t "stderr: ~s~%" errs)
	(cond ((not (= status 0))
	       (format #t "BAD non-zero status: (~s)~%" status)
	       #f)
	      ((string? expect)
	       (let ((result outs))
		 (cond ((match-to-string expect result)
			(test-loop (+ i 1) env))
		       (else
			(format #t "BAD result: ~s~%" result)
			#f))))
	      (else
	       (let ((result (with-input-from-string outs json-read)))
		 (cond ((match-to-template expect result)
			(let ((env1 (record-values env result records 0)))
			  (test-loop (+ i 1) env1)))
		       (else
			(format #t "BAD result: ~s~%" result)
			#f)))))))
     (else
      (test-loop (+ i 1) env)))))

(define (check-all tests)
  ;; Runs a list of tests.
  (let ((n (vector-length tests)))
    (let test-loop ((i 0)
		    (env '()))
      (if (< i n)
	  (let ((item (vector-ref tests i)))
	    (check-run item i env test-loop))
	  #t))))

(define (check-one id tests env)
  ;; Runs one test with a given id.  Note this cannot pass
  ;; variable-value bindings.
  (let ((n (vector-length tests)))
    (let test-loop ((i 0)
		    (env env))
      (if (< i n)
	  (let* ((item (vector-ref tests i))
		 (id1 (cond ((assoc 'ID item) => cdr) (else #f))))
	    (if (and id1 (= id id1))
		(check-run item i env (lambda (j env) #t))
		(test-loop (+ i 1) env)))
	  #t))))

(define tests (cdr (assoc 'test
			  (with-input-from-file "./artifact-s3cli.json"
			    json-read))))

(define tests (cdr (assoc 'test
			  (with-input-from-file "./artifact-argument.json"
			    json-read))))
