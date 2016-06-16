FROM thewtex/cross-compiler-linux-x64:latest
RUN apt-get -y update && apt-get -y install bc && apt-get -y clean
