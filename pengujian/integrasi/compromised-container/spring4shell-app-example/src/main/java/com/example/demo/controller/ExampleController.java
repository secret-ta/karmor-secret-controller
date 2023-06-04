package com.example.demo.controller;

import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.ui.Model;
import org.springframework.web.bind.annotation.*;
import org.springframework.web.client.RestTemplate;
import org.springframework.beans.factory.annotation.Value;

@RestController
public class ExampleController {
    @Value("${secret-v1.USERNAME}")
    private String SECRET_V1_USERNAME_PATH;

    @Value("${secret-v1.PASSWORD}")
    private String SECRET_V1_PASSWORD_PATH;

    @Value("${secret-v2.USERNAME}")
    private String SECRET_V2_USERNAME_PATH;

    @Value("${secret-v2.PASSWORD}")
    private String SECRET_V2_PASSWORD_PATH;

    @Value("${SECRET_V1_USERNAME_ENV}")
    private String SECRET_V1_USERNAME_ENV;

    @Value("${SECRET_V1_PASSWORD_ENV}")
    private String SECRET_V1_PASSWORD_ENV;

    @Value("${SECRET_V2_USERNAME_ENV}")
    private String SECRET_V2_USERNAME_ENV;

    @Value("${SECRET_V2_PASSWORD_ENV}")
    private String SECRET_V2_PASSWORD_ENV;

    @GetMapping("/log-secret")
    public String getSecrets() {
        System.out.println("=================================");
        System.out.println("MOUNTED FILE SECRET");
        System.out.println("USERNAME_V1: " + SECRET_V1_USERNAME_PATH);
        System.out.println("PASSWORD_V1: " + SECRET_V1_PASSWORD_PATH);
        System.out.println("USERNAME_V2: " + SECRET_V2_USERNAME_PATH);
        System.out.println("PASSWORD_V2: " + SECRET_V2_PASSWORD_PATH);
        System.out.println();
        System.out.println("ENVIRONMENT VARIABLES SECRET");
        System.out.println("USERNAME_V1: " + SECRET_V1_USERNAME_ENV);
        System.out.println("PASSWORD_V1: " + SECRET_V1_PASSWORD_ENV);
        System.out.println("USERNAME_V2: " + SECRET_V2_USERNAME_ENV);
        System.out.println("PASSWORD_V2: " + SECRET_V2_PASSWORD_ENV);
        System.out.println("=================================");

        return "success";
    }

    @GetMapping("/greeting")
    public String greetingForm(Model model) {
        model.addAttribute("greeting", new Greeting());
        return "hello";
    }

    @PostMapping("/greeting")
    public String greetingSubmit(@ModelAttribute Greeting greeting, Model model) {
        return "hello";
    }

    @GetMapping("/kill")
    public void kill() {
        System.exit(0);
    }
}
