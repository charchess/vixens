# This is a deliberately static, reviewed mirror of the exact upstream Go
# builder used by the TXO Fabric operator. Do not replace the digest with a tag.
FROM mirror.gcr.io/library/golang@sha256:921f61120011c774a7b37fd80bd3c3d259636090d3cd002b6a344322a0328082

LABEL org.opencontainers.image.source="https://github.com/charchess/vixens"
LABEL org.opencontainers.image.description="Pinned TXO Fabric Go builder mirror"
LABEL org.opencontainers.image.base.name="mirror.gcr.io/library/golang@sha256:d4c4845f5d60c6a974c6000ce58ae079328d03ab7f721a0734277e69905473e5"
