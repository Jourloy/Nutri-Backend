FROM scratch

WORKDIR /

COPY . .

CMD [ "./server" ]
