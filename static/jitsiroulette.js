var uuid = "";

function init() {
   var req = new XMLHttpRequest();
   req.open("POST", "/api/register", true);
   req.onreadystatechange = function () {
       if (this.readyState == 4 && this.status == 200) {
           uuid = req.responseText;
           console.log(uuid);
       }
   }

   req.send(null);
}

function poll() {
   var req = new XMLHttpRequest();
   req.open("POST", "/api/poll", true);
   req.setRequestHeader( 'Content-Type', 'application/x-www-form-urlencoded' );
   req.onreadystatechange = function () {
       if (this.readyState == 4 && this.status == 200) {
           if (req.responseText == "wait") {
              return;
           } else {
            window.location = req.responseText;
           }
       }
   }

   req.send("uuid="+uuid);
}

function wait(timeout) {
   return new Promise(resolve => {
       setTimeout(resolve, timeout);
   });
}
async function main() {
   init();
   while(true) {
      await wait(5000);
      poll();
   }
}

main();