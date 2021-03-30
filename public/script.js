let DOMSTR = '<"top"lp>rt<"bottom"i><"clear">';

// max number of pages to display:
$.fn.DataTable.ext.pager.numbers_length = 5;

// truncate long strings:
$.fn.dataTable.render.ellipsis = function (cutoff, wordbreak, escapeHtml) {
  var esc = function (t) {
    return t
      .replace(/&/g, '&amp;')
      .replace(/</g, '&lt;')
      .replace(/>/g, '&gt;')
      .replace(/"/g, '&quot;');
  };

  return function (d, type, row) {
    // Order, search and type get the original data
    if (type !== 'display') {
      return d;
    }

    if (typeof d !== 'number' && typeof d !== 'string') {
      return d;
    }

    d = d.toString(); // cast numbers

    if (d.length < cutoff) {
      return d;
    }

    var shortened = d.substr(0, cutoff - 1);

    // Find the last white space character in the string
    if (wordbreak) {
      shortened = shortened.replace(/\s([^\s]*)$/, '');
    }

    // Protect against uncontrolled HTML input
    if (escapeHtml) {
      shortened = esc(shortened);
    }

    return '<span class="ellipsis" title="' + esc(d) + '">' + shortened + '&#8230;</span>';
  };
};

$(document).ready(function () {
  $('#default').DataTable({
    ordering: false,
    columns: [{
      // width: "20%",
      render: $.fn.dataTable.render.ellipsis(40)
    }],
    dom: DOMSTR,
    // pageLength: 25
  });

  $('#search').on('keyup click', function () {
    let tables = $('.dataTable').DataTable();
    tables.tables().search($(this).val()).draw();
  });
});

var ws = new WebSocket("ws://localhost:81/admin")

ws.onmessage = function (event) {
  let msg = JSON.parse(event.data);

  switch (msg.Type) {
    case "clientEvent":
      log("connected clients: " + msg.Values[0]);
      document.getElementById("clients").textContent = msg.Values[0];
      break;
    case "dataEvent":
      log("data event: " + msg.Command + ", values: " + msg.Values.join(', '));
      processDataEvent(msg);
      break;
    case "csvSync":
      log("sync event: " + msg.Type);
      processSyncEvent(msg);
      break;
    case "download":
      log("download event");
      processDownloadEvent(msg);
      break;
  }
}

function processDataEvent(msg) {
  switch (msg.Command) {
    case "addTop":
      addTop(msg.Key, msg.Values);
      break;
    case "addBottom":
      addBottom(msg.Key, msg.Values);
      break;
    case "removeTop":
      removeTop(msg.Key);
      break;
    case "removeBottom":
      removeBottom(msg.Key);
      break;
  }
}

function processSyncEvent(msg) {
  // delete existing CSV tables
  $('.csvTable').DataTable().destroy();
  $('.csvTable').remove();

  switch (msg.Type) {
    case "csvSync":
      for (var key in msg.Data) {
        addBottom(key, msg.Data[key]);
      }
      break;
    case "kvpSync":
      // TODO
      break;
  }
}

function processDownloadEvent(msg) {
  let element = document.createElement('a');
  element.setAttribute('href', 'data:text/plain;charset=utf-8,' + encodeURIComponent(msg.Value));
  element.setAttribute('download', `sds_${new Date().getTime()}.json`);

  element.style.display = 'none';
  document.body.appendChild(element);
  element.click();
  document.body.removeChild(element);
}

function addTop(key, values) {
  if (!elemExists(key)) {
    addDataTable(key);
  }

  let table = $('#' + key).DataTable();

  // adding to the top requires re-draw of all rows
  let newData = table.column(0).data().toArray();
  if (newData.length > 0) {
    //newData.unshift(values);
    newData = [...values, ...newData];
    table.clear();
  } else {
    newData = newData.concat(values);
  }

  // DataTables expects an array of arrays, where the former represents an individual row,
  // and the latter an entry for each column. As we just have a single column in each
  // DataTable, we first need to create this array of arrays...
  let rows = newData.map(function (d) {
    return [d];
  });

  table.rows.add(rows).draw();
}

function addBottom(key, values) {
  if (!elemExists(key)) {
    addDataTable(key);
  }

  let table = $('#' + key).DataTable();

  let rows = values.map(function (d) {
    return [d];
  });

  table.rows.add(rows).draw();
}

function removeTop(key) {
  if (!elemExists(key)) {
    // should never happen!
    console.error(`No data found for key '${key}'; data out-of-sync!`);
  }

  let table = $('#' + key).DataTable();
  let row = table.row(0);
  val = row.data();
  row.remove().draw();
}

function removeBottom(key) {
  if (!elemExists(key)) {
    // should never happen!
    console.error(`No data found for key '${key}'; data out-of-sync!`);
  }

  let table = $('#' + key).DataTable();
  let row = table.row(':last');
  let val = row.data();
  row.remove().draw();
}

function elemExists(key) {
  return (document.getElementById(key) != null);
}

// creates a new data table for the specified 'key'
function addDataTable(key) {
  // remove the default table if it exists
  if (elemExists('default')) {
    $('#default').DataTable().destroy();
    $('#default').remove();
  }

  // enable saving
  $('#save').removeClass('disabled');

  $('<table><thead><tr><th>' + key + '</th></tr></thead></table>')
    .attr('id', key)
    .addClass('table table-striped compact csvTable')
    .appendTo('#dataTables');

  $('#' + key).DataTable({
    ordering: false,
    columns: [{
      name: key,
      // width: "20%",
      render: $.fn.dataTable.render.ellipsis(40)
    }],
    dom: DOMSTR,
    // pageLength: 25
  });
}

const loadBtn = document.querySelector('#file-input');
loadBtn.addEventListener('input', load);

function load() {
  let file = document.querySelector('#file-input').files[0];
  let fr = new FileReader();

  fr.addEventListener("load", e => {
    let data = JSON.parse(fr.result);
    console.log(data);
    ws.send(JSON.stringify({
      Command: "upload",
      Value: JSON.stringify(data)
    }))
  });
  
  fr.readAsText(file);
}

function save() {
  // save asks the server for the current dataStore
  // download handled by processDownloadEvent
  ws.send(JSON.stringify({
    Command: "download"
  }));
}

function log(msg) {
  let logElem = document.getElementById("log");
  let entry = document.createElement("div");
  entry.classList.add('logEntry');
  let isScrolledToBottom = logElem.scrollHeight - logElem.clientHeight <= logElem.scrollTop + 1;

  entry.innerHTML = msg;
  logElem.appendChild(entry);

  if (isScrolledToBottom) {
    logElem.scrollTop = logElem.scrollHeight - logElem.clientHeight;
  }
}