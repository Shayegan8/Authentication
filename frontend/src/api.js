export async function pre_get(endpoint) {
    const getRandomToken = await fetch("http://127.0.0.1:1234/token")
    const randomTokenWithStuff = await getRandomToken.text()
    const nextFetch = await fetch(`http://127.0.0.1:1235/${endpoint}`, {
        headers: {
            "sideline": randomTokenWithStuff
        },
        credentials: "include"
    })
    return await nextFetch.text()
}

export async function register() { // this throws a menu in the same url and the background gets shadowed and darker except the menu itself and the menu has the masterImage and titleImage and user should give the answer with the submit button
    const ctrf_token = await pre_get("register")
    const registerFetch = await fetch("http://127.0.0.1:1235/register", {
        headers: {
            "csrf-token": ctrf_token
        },
        method: "POST",
        credentials: "include"
    })
    const registerJson = await registerFetch.json()
    console.log(registerJson)
    return { titleImage: registerJson.titleImage, masterImage: registerJson.masterImage, dx: registerJson.dx, dy: registerJson.dy, width: registerJson.width, height: registerJson.height, status: registerFetch.status }
}

export async function registerValidation(email, username, password, x) { // this is the validation process of the answer and if it was true, important thing to know is we are still in /register url
    const ctrf_token = await pre_get("register/validation")
    const registerFetch = await fetch("http://127.0.0.1:1235/register/validation", {
        headers: {
            "csrf-token": ctrf_token,
            "captchaAnswer": `{"x":"${x}"}`,
            "email": email,
            "username": username,
            "password": password,
        },
        method: "POST",
        credentials: "include"
    })
    return registerFetch.status != 202
}

export async function registerValidationJWT() { // now we got redirected from /register to /register/validation/jwt
    const ctrf_token = await pre_get("register/validation/jwt")
    const registerFetch = await fetch("http://127.0.0.1:1235/register/validation/jwt", {
        headers: {
            "csrf-token": ctrf_token,
        },
        method: "POST",
        credentials: "include"
    })
    return registerFetch.status != 202
}

export async function registerValidationSubmit(vcode) { // we are still in /register/validation/jwt
    const ctrf_token = await pre_get("register/validation/submit")
    const registerFetch = await fetch("http://127.0.0.1:1235/register/validation/submit", {
        headers: {
            "csrf-token": ctrf_token,
            "verification": vcode,
        },
        method: "POST",
        credentials: "include"
    })

    return registerFetch.status != 202
}