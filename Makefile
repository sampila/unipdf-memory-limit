docker-run:
	docker run -m 1GiB\
	 --mount type=bind,source=${PWD},target=/app/\
	  -w /app/\
      --entrypoint /bin/sh\
      unipdf-memory-test\
      run_test.sh