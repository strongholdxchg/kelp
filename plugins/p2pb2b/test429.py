import sys
from datetime import datetime
from subprocess import Popen, PIPE
from json import loads
from time import sleep


def testrun():
    host = sys.argv[1]

    key = "b3ac87f45c1e7818ae47aa247905f6dd"
    nonce = "1553652297"
    payload = "eyJyZXF1ZXN0IjoiL2FwaS92MS9hY2NvdW50L2JhbGFuY2VzIiwibm9uY2UiOjE1NTM2NTIyOTd9"
    sig = "265a96d7de1c932828a7187248eef2954d6a53e9c68450d2353cccf823becba4cd5929aca160cd65893e65859f93e282975e379c1d029f94beadc199b015cb89"

    test = "curl --location --request POST --header \"Content-Type: application/json\" " \
           "--header \"X-TXC-APIKEY: " + key +"\" " \
           "--header \"X-TXC-PAYLOAD: " + payload +"\" " \
           "--header \"X-TXC-SIGNATURE: " + sig + "\" " \
           "--data '{\"request\":\"/api/v1/account/balances\",\"nonce\":" + nonce +"}' " \
           "--url " + host + "/api/v1/account/balances 2>/dev/null"

    result_ = Popen(test, shell=True, stdout=PIPE).stdout.read().decode()

    # the first marvel is that it gives JSON response twice
    lenr = len(result_)
    if lenr % 2 == 0:
        lenr = int(lenr / 2)
        slice1 = result_[0:lenr]
        slice2 = result_[lenr:]
        if slice1 == slice2:
            result_ = slice1
    result = loads(result_)
    print(result)

    # absence of error is given by a empty message string
    error = result['message'] != ""
    print(error)

    return error

i = 1
while True:
    print(datetime.now().timestamp(), " ", i)
    error = testrun()
    if error:
        sys.exit(error)

    sleep(0.2)
    i += 1
