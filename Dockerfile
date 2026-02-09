FROM ubuntu:latest

# Install necessary packages
RUN apt-get update && apt-get install -y \
    bash wget vim curl git ca-certificates && \
    rm -rf /var/lib/apt/lists/*

# Install Go using webi and force it into the system PATH
RUN curl -sS https://webi.sh/golang | sh
ENV PATH="/root/.local/bin:/root/.config/envman/PATH.env:$PATH"
# Manually add typical Go bin paths to the ENV so Docker sees them
ENV PATH="/root/go/bin:/usr/local/go/bin:$PATH"

# Install Starship
RUN curl -sS https://starship.rs/install.sh | sh -s -- -y && \
    echo 'eval "$(starship init bash)"' >> ~/.bashrc

# Install Tools (Goose, SQLC, BootDev)
RUN . ~/.config/envman/PATH.env && \
	go install github.com/bootdotdev/bootdev@latest && \
    go install github.com/pressly/goose/v3/cmd/goose@latest && \
    go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest

WORKDIR /gator

CMD ["/bin/bash"]

