import ws from 'k6/ws';
import { sleep } from 'k6';

export default function () {
  var url = 'ws://localhost:81/connect';

  populateData(url);
}

function populateData(url) {
  ws.connect(url, null, function (socket) {
    socket.on('open', function () {
      console.log("WebSocket connection established.");

      addTop(socket, 'key1', new Date().getTime().toString() + 'somereallylongvaluesomereallylongvaluesomereallylongvaluesomereallylongvaluesomereallylongvaluesomereallylongvalue');
      addBottom(socket, 'key2', new Date().getTime().toString());
      addTop(socket, 'key3', new Date().getTime().toString());
      addBottom(socket, 'key4', new Date().getTime().toString());
      addTop(socket, 'key5', new Date().getTime().toString());
      addBottom(socket, 'key6', new Date().getTime().toString());
      close(socket);
    });
  });
}

function close(socket) {
  socket.close();
  console.log("WebSocket connection closed.");
}

function addTop(socket, key, value) {
  socket.send(JSON.stringify(
    {
      command: 'addTop',
      key: key,
      values: [
        value
      ]
    }
  ))
}

function addBottom(socket, key, value) {
  socket.send(JSON.stringify(
    {
      command: 'addBottom',
      key: key,
      values: [
        value
      ]
    }
  ));
}
