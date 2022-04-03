FROM debian:11-slim

LABEL "org.opencontainers.image.authors"="KeisukeFD"
LABEL "org.opencontainers.image.description"="Packaging GoBackup (rclone and restic) \
https://github.com/KeisukeFD/gobackup"

ARG restic_version=0.13.0
ARG rclone_version=1.58.0
ARG gobackup_version=1.0.2

RUN apt-get update && apt-get install -y \
  curl \
  bzip2 \
  unzip \
  && rm -rf /var/lib/apt/lists/*

RUN curl -L "https://github.com/restic/restic/releases/download/v${restic_version}/restic_${restic_version}_linux_amd64.bz2" -o /tmp/restic.bz2 \
    && cd /tmp \
    && bzip2 -d restic.bz2 \
    && mv /tmp/restic /usr/bin/restic \
    && chmod +x /usr/bin/restic

RUN curl -L "https://github.com/rclone/rclone/releases/download/v${rclone_version}/rclone-v${rclone_version}-linux-amd64.zip" -o /tmp/rclone.zip \
    && cd /tmp \
    && unzip -p rclone.zip rclone-v${rclone_version}-linux-amd64/rclone > rclone \
    && mv /tmp/rclone /usr/bin/rclone \
    && chmod +x /usr/bin/rclone

RUN curl -L "https://github.com/KeisukeFD/gobackup/releases/download/v${gobackup_version}/gobackup_${gobackup_version}_linux_amd64.tar.gz" -o /tmp/gobackup.tar.gz \
    && cd /tmp \
    && tar -xf gobackup.tar.gz \
    && rm /tmp/gobackup.tar.gz \
    && mv /tmp/gobackup /usr/bin/gobackup \
    && chmod +x /usr/bin/gobackup

VOLUME /backup

WORKDIR /app

ENTRYPOINT ["gobackup"]
CMD ["-c /app/config.yml", "backup"]
