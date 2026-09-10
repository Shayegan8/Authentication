export async function register() { // this throws a menu in the same url and the background gets shadowed and darker except the menu itself and the menu has the masterImage and titleImage and user should give the answer with the submit button
    const registerFetch = await fetch("http://127.0.0.1:1235/register", {
        method: "POST",
        credentials: "include"
    })
    const registerJson = await registerFetch.json()
    console.log(registerJson)
    return { titleImage: registerJson.titleImage, masterImage: registerJson.masterImage, dx: registerJson.dx, dy: registerJson.dy, width: registerJson.width, height: registerJson.height, status: registerFetch.status }
}

export async function registerValidation(email, username, password, x) { // this is the validation process of the answer and if it was true, important thing to know is we are still in /register url
    const registerFetch = await fetch("http://127.0.0.1:1235/register/validation", {
        headers: {
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
    const registerFetch = await fetch("http://127.0.0.1:1235/register/validation/jwt", {
        method: "POST",
        credentials: "include"
    })
    return registerFetch.status != 202
}

export async function registerValidationSubmit(vcode) { // we are still in /register/validation/jwt
    const registerFetch = await fetch("http://127.0.0.1:1235/register/validation/submit", {
        headers: {
            "verification": vcode,
        },
        method: "POST",
        credentials: "include"
    })

    return registerFetch.status != 202
}

export async function login() { // this throws a menu in the same url and the background gets shadowed and darker except the menu itself and the menu has the masterImage and titleImage and user should give the answer with the submit button
    const registerFetch = await fetch("http://127.0.0.1:1235/login", {
        method: "POST",
        credentials: "include"
    })
    const registerJson = await registerFetch.json()
    console.log(registerJson)
    return { titleImage: registerJson.titleImage, masterImage: registerJson.masterImage, dx: registerJson.dx, dy: registerJson.dy, width: registerJson.width, height: registerJson.height, status: registerFetch.status }
}

export async function loginValidation(email, password, x) { // this is the validation process of the answer and if it was true, important thing to know is we are still in /register url
    const registerFetch = await fetch("http://127.0.0.1:1235/login/validation", {
        headers: {
            "captchaAnswer": `{"x":"${x}"}`,
            "email": email,
            "password": password,
        },
        method: "POST",
        credentials: "include"
    })
    return registerFetch.status != 202
}

export async function loginValidationJWT() { // now we got redirected from /register to /register/validation/jwt
    const registerFetch = await fetch("http://127.0.0.1:1235/login/validation/jwt", {
        method: "POST",
        credentials: "include"
    })
    return registerFetch.status != 202
}

export async function loginValidationSubmit(vcode) { // we are still in /register/validation/jwt
    const registerFetch = await fetch("http://127.0.0.1:1235/login/validation/submit", {
        headers: {
            "verification": vcode,
        },
        method: "POST",
        credentials: "include"
    })

    return registerFetch.status != 202
}