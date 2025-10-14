FROM scratch

WORKDIR /

COPY . .

EXPOSE 3002

CMD [ "./server" ]
