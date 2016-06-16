FROM thewtex/cross-compiler-linux-armv7:latest
RUN apt-get -y update && apt-get -y install bc && apt-get -y clean
