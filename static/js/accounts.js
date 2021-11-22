function createAccount(ac) {
    let acbox = document.createElement('a');
    acbox.setAttribute('class', 'account-panel');
    acbox.setAttribute('data-usertype', ac.user_type);

    let iconDisp = document.createElement("div");
    iconDisp.setAttribute("class", "icon-disp");
    iconDisp.style.backgroundImage = "url('/Account/img/" + ac.id + "')";
    acbox.appendChild(iconDisp);

    let detailsArea = document.createElement('div');
    acbox.appendChild(detailsArea);

    let namelabel = document.createElement("div");
    namelabel.innerText = ac.name;
    detailsArea.appendChild(namelabel);

    let desc = document.createElement('div');
    desc.innerText = ac.description;
    detailsArea.appendChild(desc);

    let lans = document.createElement('div');
    lans.innerText = ac.langs.map(l => l.lang).join(', ');
    detailsArea.appendChild(lans);

    acbox.href = '/u/' + ac.id;

    return acbox;
}