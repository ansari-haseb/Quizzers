FROM python:3.9-alpine
ADD main.py .
ADD requirements.txt .
COPY html/* /html/
RUN pip install beautifulsoup4 python-arango Flask tablib[html] --no-cache-dir -r requirements.txt
CMD python ./main.py