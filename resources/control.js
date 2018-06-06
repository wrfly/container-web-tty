// function container_Start(cid) {
// console.info("start container "+cid)
// }

try {
    var htmlBtns = document.getElementsByTagName('button');
    for (var i = 0; i < htmlBtns.length; ++i) {
        htmlBtns[i].onclick = function () {
            var cid = this.parentElement.parentElement.querySelector('a').getAttribute('title');
            var action = this.title;
            alert(action + cid);
        };
    }
} catch (error) {
    alert(error);
}