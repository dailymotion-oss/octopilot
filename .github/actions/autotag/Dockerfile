ARG autotag_version=v1
FROM quay.io/pantheon-public/autotag:${autotag_version}

LABEL "com.github.actions.name"="autotag"
LABEL "com.github.actions.description"="Wraps https://github.com/pantheon-systems/autotag as a GitHub action"
LABEL "com.github.actions.icon"="tag"
LABEL "com.github.actions.color"="blue"

COPY entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

ENTRYPOINT ["/entrypoint.sh"]