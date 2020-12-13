var pageHeader = document.createElement("header");
pageHeader.setAttribute("class", "page-header");

var serviceTitle = document.createElement("div");
serviceTitle.setAttribute("class", "service-title");
serviceTitle.innerText = "Live interpreting";
pageHeader.appendChild(serviceTitle);

document.body.appendChild(pageHeader);

function appendHeader(elm) {
	pageHeader.appendChild(elm);
}