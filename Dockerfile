FROM alpine:3.4

ENV HOME /home/user
RUN adduser -u 1001 -D user \
  && mkdir -p $HOME/data \
  && chown -R user:user $HOME

COPY currency-exchange $HOME/
RUN wget -O $HOME/data/eurofxref-hist.xml http://www.ecb.europa.eu/stats/eurofxref/eurofxref-hist.xml

WORKDIR $HOME
VOLUME $HOME/data

USER user
EXPOSE 8080
CMD ["./currency-exchange", "-historic.data", "data/eurofxref-hist.xml", "-http.addr", ":8080"]
