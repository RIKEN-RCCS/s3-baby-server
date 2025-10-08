;; check.scm

;; This is for "guile --r7rs", 3.0.9 and later.

(import
 (ice-9 exceptions)
 (ice-9 binary-ports)
 (ice-9 textual-ports)
 (ice-9 expect)
 (ice-9 popen)
 (ice-9 format)
 (ice-9 match)
 (scheme base)
 (srfi srfi-1) ;; list
 (srfi srfi-11) ;; multiple-values
 ;;(srfi srfi-28) ;; format
 (only (srfi srfi-43) vector-copy) ;; vector-library
 (srfi srfi-60) ;; integers as bits
 (only (rnrs base) assert))

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

(define time-regexp
  (string-append
   "^"
   "[0-9]{4}-[0-9]{2}-[0-9]{2}" ;; "2025-08-20"
   "T"
   "[0-9]{2}:[0-9]{2}:[0-9]{2}\\.[0-9]{6}" ;; "08:32:06.081000"
   "+"
   "[0-9]{2}:[0-9]{2}" ;; "00:00"
   "$"))

(define (expect-regexp? expect)
  ;; Checks an expect string is for an regexp, i.e., beginning with
  ;; "#|".  It returns a pattern for a whole string ("^regexp$").
  (cond ((string-match "^#\\|(.*)$" expect)
	 => (lambda (s)
	      (string-append "^" (match:substring s 1) "$")))
	(else #f)))

(define (match-to-string expect result)
  ;; Matches a line (prefix) of the result.
  (string-match (string-append "^" expect "\n") result))

(define (match-to-template expect result)
  ;; Matches a result to an expected, both in json.  It return a
  ;; boolean.  A wilecard string ("#_") may match any entity.  Other
  ;; strings match strings.  "#time" is a time string, and "#|regexp"
  ;; is a string by regexp.  A vector pattern be singleton or empty.
  ;; An empty vector matches any vector.  An object pattern accepts
  ;; excess entities.
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
	   (cond ((string=? expect "#time")
		  (string-match time-regexp result))
		 ((expect-regexp? expect)
		  => (lambda (regexp)
		       (string-match regexp result)))
		 (else (string=? expect result))))
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
		  #t)
		 ((= (vector-length expect) 1)
		  (let ((expect1 (vector-ref expect 1))
			(n (vector-length result)))
		    (let loop ((i 0))
		      (if (< i n)
			  (let ((result1 (vector-ref result 1)))
			    (if (match-to-template expect1 result1)
				(loop (+ i 1))
				#f))
			  #t))))))
	  (else #f)))
   (else
    (format #t "BAD template: expect=~s~%" expect)
    #f)))

(define (read-output-string s)
  (with-input-from-string s json-read))

(define (check-all tests)
  (let ((n (vector-length tests)))
    (let loop ((i 0))
      (if (< i n)
	  (let* ((item (vector-ref tests i))
		 (o (assoc 'operation item))
		 (expect (cdr (assoc 'outcome1 item))))
	    (when (not (eq? o #f))
	      (format #t "testing: ~s ~s~%"
		      (assoc 'name item)
		      (assoc 'kind item))
	      (format #t "expect: ~s~%" expect)
	      (format #t "operation: ~s~%"
		      (cdr o))
	      (let-values (((status outs errs) (run-system (cdr o))))
		(format #t "status: ~s~%" status)
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
	    (loop (+ i 1)))
	  #t))))

(define tests (cdr (assoc 'test
			  (with-input-from-file "./artifact-short.json"
			    json-read))))
