let base = document.createElement('div');

let pageHeader = document.createElement("header");
pageHeader.setAttribute("class", "page-header");

let innerHeader = document.createElement('div');
innerHeader.setAttribute('class', 'inner-header');
pageHeader.appendChild(innerHeader);

let humb = document.createElement('label');
humb.setAttribute('class', 'header-humb');
humb.setAttribute('for', 'humbCheck');
innerHeader.appendChild(humb);

let humbCheck = document.createElement('input');
humbCheck.id = 'humbCheck';
humbCheck.style.display = 'none';
humbCheck.setAttribute('type', 'checkbox');
humb.appendChild(humbCheck);

let serviceTitle = document.createElement("div");
serviceTitle.setAttribute("class", "service-title");
innerHeader.appendChild(serviceTitle);

let logoImg = document.createElement('svg');
logoImg.style.width = "100%";
logoImg.style.height = "100%";
logoImg.setAttribute('fill', 'white');
let use = document.createElement('use');
use.setAttribute('xlink:href', '/st/materials/logo.svg#logo');
logoImg.appendChild(use);
serviceTitle.appendChild(logoImg);

base.appendChild(pageHeader);
document.write(base.innerHTML);

document.getElementById('humbCheck').addEventListener('change', elm => {
	let sidemenu = document.getElementById('sidemenu');
	if (sidemenu == null) return;
	if (elm.target.checked) sidemenu.style.height = 'auto';
	else sidemenu.removeAttribute('style');
});

function appendHeader(elm) {
	document.querySelector('.inner-header').appendChild(elm);
}