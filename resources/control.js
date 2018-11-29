// container control

try {
    var htmlBtns = document.getElementsByTagName('button');
    for (var i = 0; i < htmlBtns.length; ++i) {
        htmlBtns[i].onclick = function () {
            var cid = this.parentElement.parentElement.querySelector('a').getAttribute('value');
            var action = this.title;
            var u = "/container/" + action + "/" + cid;
            var xmlhttp = new XMLHttpRequest();
            xmlhttp.open("POST", u);
            xmlhttp.onreadystatechange = function () {
                if (xmlhttp.readyState == 4) {
                    var j = JSON.parse(xmlhttp.responseText);
                    console.debug(j);
                    if (xmlhttp.status != 200) {
                        alert(xmlhttp.responseText);
                    }
                }
            };
            alert(action + " container " + cid.substring(0, 8));
            console.debug("POST: " + u);
            xmlhttp.send();
        };
    }
} catch (error) {
    console.error(error);
}