FROM ubuntu:latest

# Install necessary packages
RUN apt-get update && apt-get install -y \
    bash \
    wget \
    vim \
    curl

# Install Go
RUN curl -sS https://webi.sh/golang | sh && \
    . ~/.config/envman/PATH.env && \
    go version

# Install Starship
RUN curl -sS https://starship.rs/install.sh > get-starship.sh
RUN chmod +x get-starship.sh 
RUN sh get-starship.sh -y
RUN echo 'eval "$(starship init bash)"' >> ~/.bashrc

# Install BootDev CLI
RUN . ~/.config/envman/PATH.env && \
    go install github.com/bootdotdev/bootdev@latest && \
    echo 'export PATH=$PATH:$HOME/go/bin' >> ~/.bashrc

# Install PostgreSQL
RUN apt-get install -y postgresql postgresql-contrib sudo

# Install Goose
RUN go install github.com/pressly/goose/v3/cmd/goose@latest

# Install SQLC
RUN go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest

# Set the working directory
WORKDIR /gator

CMD ["/bin/bash"]
