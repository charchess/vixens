# This is a deliberately static, reviewed mirror of the exact upstream Go
# builder used by the TXO Fabric operator. Do not replace the digest with a tag.
FROM mirror.gcr.io/library/golang@sha256:2920bb124f1a2e12a1d39f0cab756ae0068441ee1da7655dec7f5e374944095b

LABEL org.opencontainers.image.source="https://github.com/charchess/vixens"
LABEL org.opencontainers.image.description="Pinned TXO Fabric Go builder mirror"
LABEL org.opencontainers.image.base.name="mirror.gcr.io/library/golang@sha256:d4c4845f5d60c6a974c6000ce58ae079328d03ab7f721a0734277e69905473e5"
