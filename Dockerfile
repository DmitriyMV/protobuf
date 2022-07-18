# syntax = docker/dockerfile-upstream:1.2.0-labs

# THIS FILE WAS AUTOMATICALLY GENERATED, PLEASE DO NOT EDIT.
#
# Generated on 2022-07-18T19:24:53Z by kres latest.

ARG TOOLCHAIN

# cleaned up specs and compiled versions
FROM scratch AS generate

# base toolchain image
FROM ${TOOLCHAIN} AS toolchain
RUN apk --update --no-cache add bash curl build-base protoc protobuf-dev

# build tools
FROM toolchain AS tools
ENV GO111MODULE on
ENV CGO_ENABLED 0
ENV GOPATH /go
ARG GOLANGCILINT_VERSION
RUN curl -sfL https://raw.githubusercontent.com/golangci/golangci-lint/${GOLANGCILINT_VERSION}/install.sh | bash -s -- -b /bin ${GOLANGCILINT_VERSION}
ARG GOFUMPT_VERSION
RUN go install mvdan.cc/gofumpt@${GOFUMPT_VERSION} \
	&& mv /go/bin/gofumpt /bin/gofumpt
ARG GOIMPORTS_VERSION
RUN go install golang.org/x/tools/cmd/goimports@${GOIMPORTS_VERSION} \
	&& mv /go/bin/goimports /bin/goimports
ARG DEEPCOPY_VERSION
RUN go install github.com/siderolabs/deep-copy@${DEEPCOPY_VERSION} \
	&& mv /go/bin/deep-copy /bin/deep-copy

# tools and sources
FROM tools AS base
WORKDIR /src
COPY ./go.mod .
COPY ./go.sum .
RUN --mount=type=cache,target=/go/pkg go mod download
RUN --mount=type=cache,target=/go/pkg go mod verify
COPY ./array_test.go ./array_test.go
COPY ./benchmarks_test.go ./benchmarks_test.go
COPY ./constructor_test.go ./constructor_test.go
COPY ./crashers_test.go ./crashers_test.go
COPY ./decode.go ./decode.go
COPY ./doc.go ./doc.go
COPY ./encode.go ./encode.go
COPY ./encoding_test.go ./encoding_test.go
COPY ./field.go ./field.go
COPY ./field_test.go ./field_test.go
COPY ./generated_message_test.go ./generated_message_test.go
COPY ./interface.go ./interface.go
COPY ./interface_test.go ./interface_test.go
COPY ./map_test.go ./map_test.go
COPY ./marshal_test.go ./marshal_test.go
COPY ./optionalBytes_test.go ./optionalBytes_test.go
COPY ./person_test.go ./person_test.go
COPY ./privateFields_test.go ./privateFields_test.go
COPY ./protobuf_test.go ./protobuf_test.go
COPY ./test1_test.go ./test1_test.go
COPY ./test2_test.go ./test2_test.go
COPY ./test3_test.go ./test3_test.go
COPY ./test_exports_test.go ./test_exports_test.go
RUN --mount=type=cache,target=/go/pkg go list -mod=readonly all >/dev/null

# runs gofumpt
FROM base AS lint-gofumpt
RUN FILES="$(gofumpt -l .)" && test -z "${FILES}" || (echo -e "Source code is not formatted with 'gofumpt -w .':\n${FILES}"; exit 1)

# runs goimports
FROM base AS lint-goimports
RUN FILES="$(goimports -l -local github.com/DmitriyMV/protobuf .)" && test -z "${FILES}" || (echo -e "Source code is not formatted with 'goimports -w -local github.com/DmitriyMV/protobuf .':\n${FILES}"; exit 1)

# runs golangci-lint
FROM base AS lint-golangci-lint
COPY .golangci.yml .
ENV GOGC 50
RUN --mount=type=cache,target=/root/.cache/go-build --mount=type=cache,target=/root/.cache/golangci-lint --mount=type=cache,target=/go/pkg golangci-lint run --config .golangci.yml

# runs unit-tests with race detector
FROM base AS unit-tests-race
ARG TESTPKGS
RUN --mount=type=cache,target=/root/.cache/go-build --mount=type=cache,target=/go/pkg --mount=type=cache,target=/tmp CGO_ENABLED=1 go test -v -race -count 1 ${TESTPKGS}

# runs unit-tests
FROM base AS unit-tests-run
ARG TESTPKGS
RUN --mount=type=cache,target=/root/.cache/go-build --mount=type=cache,target=/go/pkg --mount=type=cache,target=/tmp go test -v -covermode=atomic -coverprofile=coverage.txt -coverpkg=${TESTPKGS} -count 1 ${TESTPKGS}

FROM scratch AS unit-tests
COPY --from=unit-tests-run /src/coverage.txt /coverage.txt

