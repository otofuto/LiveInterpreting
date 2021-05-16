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

humbCheck.addEventListener('change', elm => {
	let sidemenu = document.getElementById('sidemenu');
	if (sidemenu == null) return;
	if (elm.target.checked) sidemenu.style.height = 'auto';
	else sidemenu.removeAttribute('style');
});

let serviceTitle = document.createElement("div");
serviceTitle.setAttribute("class", "service-title");
innerHeader.appendChild(serviceTitle);

let logoImg = document.createElement('img');
logoImg.src = '/st/materials/logo.svg#logo';
serviceTitle.appendChild(logoImg);

document.body.appendChild(pageHeader);

function appendHeader(elm) {
	innerHeader.appendChild(elm);
}