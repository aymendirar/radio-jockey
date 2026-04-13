FROM ruby:4.0.2-alpine AS builder

RUN apk add --no-cache build-base libsodium-dev opus-dev git

WORKDIR /client

COPY Gemfile Gemfile.lock ./
RUN bundle config set --local without 'development test' && bundle install

COPY . ./

FROM ruby:4.0.2-alpine

RUN apk add --no-cache libsodium-dev opus-dev

WORKDIR /client

COPY --from=builder /usr/local/bundle /usr/local/bundle
COPY --from=builder /client ./

CMD ["bundle", "exec", "ruby", "-I", "src/proto", "src/main.rb"]
