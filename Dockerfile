# The ghcr.io/joshuasprow/nitrovin image (ADR 0007), built by goreleaser's
# dockers_v2 from the binaries it already built: the context holds
# <os>/<arch>/nitrovin for each platform. The binary is static and carries its
# data, so the image is the binary alone, run as a non-root user.
#
# /tmp is that user's, empty and mode 0700: the scratch `vocab compile` sorts
# in, for a build stage FROM this image that compiles a catalog
# (docs/vocabularies.md, "At image build"). Serving writes no files, so only
# the image's user needs it, and 1777 is not to be had: COPY --chmod drops
# the sticky bit, leaving 0777. It is copied from the empty stage below: the
# final image has no shell to make it.
FROM scratch AS empty

FROM scratch
ARG TARGETPLATFORM
COPY LICENSE NOTICE /usr/share/doc/nitrovin/
COPY $TARGETPLATFORM/nitrovin /usr/bin/nitrovin
COPY --from=empty --chown=65532:65532 --chmod=0700 / /tmp/
USER 65532:65532
EXPOSE 8080
ENTRYPOINT ["/usr/bin/nitrovin"]
CMD ["serve", "--addr", ":8080"]
