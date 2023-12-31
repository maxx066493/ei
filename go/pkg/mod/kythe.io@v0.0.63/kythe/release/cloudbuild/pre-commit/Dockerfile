# Copyright 2020 The Kythe Authors. All rights reserved.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#   http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# docker build -t gcr.io/kythe-repo/pre-commit .
FROM ubuntu:22.04

RUN apt-get update \
        && apt-get upgrade -y \
        && apt-get install -y --no-install-recommends \
                curl ca-certificates \
                # pre-commit dependencies
                python3 python3-dev python3-pip \
                # Linter dependencies
                golang-1.18-go shellcheck clang-format-11 openjdk-11-jre-headless git \
        && apt-get clean \
        && rm -rf /var/lib/apt/lists/*

# Make clang-format-11 the default
RUN update-alternatives \
      --install /usr/bin/clang-format clang-format /usr/bin/clang-format-11    100

# Make go-1.18 the default
RUN update-alternatives \
      --install /usr/bin/go go /usr/lib/go-1.18/bin/go  100
RUN update-alternatives \
      --install /usr/bin/gofmt gofmt /usr/lib/go-1.18/bin/gofmt  100

# Install pip-packages
COPY requirements.txt pre-commit-requirements.txt
RUN pip3 install --require-hashes -r pre-commit-requirements.txt \
 && rm pre-commit-requirements.txt

# Install extra linters
RUN go install github.com/bazelbuild/buildtools/buildifier@latest \
 && go install golang.org/x/lint/golint@latest \
 && go install honnef.co/go/tools/cmd/staticcheck@latest

# Fetch the latest version of google-java-format from GitHub
RUN curl -s https://api.github.com/repos/google/google-java-format/releases/latest \
        | sed -n '/browser_download_url/s/[^:]*:[^"]*\("[^"]*"\).*/url = \1/p' \
        | egrep 'google-java-format-[^-]*-all-deps.jar' \
        | curl -L -o /usr/bin/google-java-format.jar -K - \
        && /bin/echo -e '#!/bin/sh\nexec java -jar /usr/bin/google-java-format.jar "$@"' >/usr/bin/google-java-format \
        && chmod +x /usr/bin/google-java-format

# Fetch the rustup installer and install the nightly standard toolchain.
# We can't rely on the default $HOME for these as they differ between container build
# ($HOME=/root) and GCB ($HOME=/builder/home).
ENV RUSTUP_HOME=/root/.rustup
ENV CARGO_HOME=/root/.cargo
RUN curl -o /tmp/rustup.sh --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs && \
        /bin/bash /tmp/rustup.sh --default-toolchain nightly-2022-07-27 --profile default -y && \
        rm /tmp/rustup.sh

# Install go wrapper script
ADD go /usr/local/bin/go
ENV PATH=$PATH:/root/go/bin:$CARGO_HOME/bin
ENTRYPOINT ["pre-commit"]
