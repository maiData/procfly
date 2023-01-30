ARG BASE_IMAGE=scratch

FROM ${BASE_IMAGE}

COPY /procfly /bin/procfly

ENV PROCFLY_DIR /etc/procfly
ENTRYPOINT [ "procfly" ]
CMD [ "run", "${PROCFLY_DIR}" ]
