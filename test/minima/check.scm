;; check.scm

;; This is for "guile --r7rs", 3.0.9 and later.

(import
 (ice-9 exceptions)
 (ice-9 binary-ports)
 (ice-9 textual-ports)
 (ice-9 expect)
 (ice-9 popen)
 (ice-9 format)
 (scheme base)
 (srfi srfi-1) ;; list
 (srfi srfi-11) ;; multiple-values
 ;;(srfi srfi-28) ;; format
 (only (srfi srfi-43) vector-copy) ;; vector-library
 (srfi srfi-60) ;; integers as bits
 (only (rnrs base) assert))

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

(define tests (with-input-from-file "./artifact-bottom.json"
		json-read))

(define (run-system command)
  ;; Returns three-values of status and strings of stdout and stderr.
  ;; It assumes a command finishes, i.e., no timeout.  (Note "spawn"
  ;; is introduced in guile-3.0.9).
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

(define (read-output-string s)
  (with-input-from-string s json-read))

(define (check-all tests)
  (let loop ((i 0))
    (let* ((item (vector-ref tests i))
	   (o (assoc 'operation item)))
      (when (not (eq? o #f))
	(format #t "testing: ~s ~s~%"
		(assoc 'name item)
		(assoc 'kind item))
	(format #t "operation: ~s~%"
		(cdr o))
	(let-values (((status outs errs) (run-system (cdr o))))
	  (format #t "stdout: ~s~%" outs)
	  (format #t "stderr: ~s~%" errs)
	  (when (not (= status 0))
	    (format #t "non-zero status~%")))))
    (if (< (+ i 1) (vector-length tests))
	(loop (+ i 1)))))
