;; check.scm

;; This is for "guile --r7rs", 3.0.9 and later.

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
 (only (scheme base) define-record-type textual-port? vector-for-each write-string)
 (only (rnrs base) infinite? assert)
 (srfi srfi-1) ;; list
 (srfi srfi-11) ;; multiple-values
 ;;(srfi srfi-28) ;; format
 ;;(only (srfi srfi-43) vector-copy) ;; vector-library
 (srfi srfi-60) ;; integers as bits
 )

;; Importing "(scheme base)" is (likely) not necessary when R7RS.

;; (import (system vm trace))
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

;; MEMO: at-at refers to a package.
;; (@@ (ice-9 popen) open-process)

(define (list-foldr f init list)
  ;; foldr : ('a * 'b -> 'b) -> 'b -> 'a list -> 'b
  (match list
    (() init)
    ((fst . rst) (f fst (list-foldr f init rst)))))

(define (list-foldl f init list)
  ;; foldl : ('a * 'b -> 'b) -> 'b -> 'a list -> 'b
  (match list
    (() init)
    ((fst . rst) (list-foldl f (f fst init) rst))))

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
  ;; finishes shortly, (it does not timeout).  (Note "spawn" is
  ;; introduced in guile-3.0.9).
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

(define (replace-in-string s token pattern)
  ;; Replaces a token with a pattern in a string s.  It replaces all
  ;; occurrences of tokens.
  (let* ((token1 (regexp-quote token)))
    (cond ((string-match token1 s)
	   => (lambda (m)
	       (let* ((range (vector-ref m 1))
		      (prefix (substring s 0 (car range)))
		      (suffix (substring s (cdr range) (string-length s))))
		 (string-append
		  prefix
		  pattern
		  (replace-in-string suffix token pattern)))))
	  (else s))))

(define (replace-regexp-token s)
  ;; Replaces pattern tokens in a string with their regexp patterns.
  ;; Tokens are "#date", "#time", "#datetime".  TOKEN PATTERNS SHOULD
  ;; BE LISTED IN LONGER TO SHORTER ORDER.
  (let ((replacements `(("#datetime" . ,datetime-regexp)
			("#date" . ,date-regexp)
			("#time" . ,time-regexp))))
    (let loop ((s s)
	       (replacements replacements))
      (if (null? replacements)
	  s
	  (let ((item (car replacements)))
	    (loop (replace-in-string s (car item) (cdr item))
		  (cdr replacements)))))))

(define (expect-regexp?~ expect)
  ;; Checks an expect string is for an regexp, i.e., beginning with
  ;; "#|".  It returns a pattern for a whole string ("^regexp$").
  (cond ((string-match "^#\\|(.*)$" expect)
	 => (lambda (s)
	      (string-append "^" (match:substring s 1) "$")))
	(else #f)))

(define (match-to-string expect result)
  ;; Matches a line (prefix) of the result.
  (string-match (string-append "^" (replace-regexp-token expect) "$") result))

(define (match-to-template~ expect result)
  (let ((v (match-to-template1 expect result)))
    (format #t "match-to-template expect=~s result=~s => ~s~%" expect result v)
    v))

(define (match-to-template expect result)
  ;; Matches a result to an expected, both in json.  It returns a
  ;; boolean.  A string pattern "#_" is a wildcard matching any
  ;; entity.  String patterns may include pattern tokens: "#time" is a
  ;; time string, etc., and "#|regexp" is a regexp pattern.  A vector
  ;; pattern should be empty or singleton.  A vector pattern ["#_"]
  ;; matches any vector including empty ones.  An object pattern
  ;; accepts excess entities.
  (cond
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

(define (read-output-string s)
  (with-input-from-string s json-read))

(define (append-string-vector v)
  ;; Appends strings in a vector with intervening newlines.
  (list-foldr (lambda (a b) (string-append a "\n" b)) "" (vector->list v)))

(define (fetch-assoc object slot)
  ;; Does assoc, but value "null" is treated as key is missing.
  (cond ((assoc slot object)
	 => (lambda (pair)
	      (if (not (eqv? (cdr pair) 'null))
		  (cdr pair)
		  #f)))
	(else #f)))

(define (fetch-outcome object)
  ;; Fetches an outcome pattern from either slot "outcome1" or
  ;; "outcome2" in an object.  An "outcome1" slot is json object, but
  ;; an "outcome2" slot is a pattern string.
  (cond ((fetch-assoc object 'outcome1)
	 => (lambda (e) e))
	((fetch-assoc object 'outcome2)
	 => (lambda (v) (append-string-vector v)))
	(else #f)))

(define (check-run item i loop)
  ;; (item = (vector-ref tests i))
  (let* ((op (assoc 'operation item))
	 (expect (fetch-outcome item))
	 (skip-flag (assoc 'skip item))
	 (stop-flag (assoc 'stop item)))
    (cond
     ((not (eqv? stop-flag #f))
      (format #t "STOP TESTING~%")
      #t)
     ((not (eqv? skip-flag #f))
      (format #t "Skipping... test ~s~%" i)
      (loop (+ i 1)))
     ((not (eqv? op #f))
      (format #t "testing: ~s ~s ~s~%"
	      (assoc 'ID item)
	      (assoc 'name item)
	      (assoc 'kind item))
      (format #t "expect: ~s~%" expect)
      (format #t "operation: ~s~%" (cdr op))
      (let-values (((status outs errs) (run-system (cdr op))))
	(format #t "status: ~s (~s, ~s)~%" status
		(status:exit-val status) (status:term-sig status))
	(format #t "stdout: ~s~%" outs)
	(format #t "stderr: ~s~%" errs)
	(cond ((not (= status 0))
	       (format #t "BAD non-zero status: (~s)~%" status)
	       #f)
	      ((string? expect)
	       (let ((result outs))
		 (if (match-to-string expect result)
		     (loop (+ i 1))
		     (begin
		       (format #t "BAD result: ~s~%" result)
		       #f))))
	      (else
	       (let ((result (with-input-from-string outs json-read)))
		 (if (match-to-template expect result)
		     (loop (+ i 1))
		     (begin
		       (format #t "BAD result: ~s~%" result)
		       #f)))))))
     (else
      (loop (+ i 1))))))

(define (check-all tests)
  (let ((n (vector-length tests)))
    (let loop ((i 0))
      (if (< i n)
	  (let ((item (vector-ref tests i)))
	    (check-run item i loop))
	  #t))))

(define (check-one id tests)
  (let ((n (vector-length tests)))
    (let loop ((i 0))
      (if (< i n)
	  (let* ((item (vector-ref tests i))
		 (id1 (cond ((assoc 'ID item) => cdr) (else #f))))
	    (if (and id1 (= id id1))
		(check-run item i (lambda (j) #t))
		(loop (+ i 1))))
	  #t))))

(define tests (cdr (assoc 'test
			  (with-input-from-file "./artifact-s3cli.json"
			    json-read))))
