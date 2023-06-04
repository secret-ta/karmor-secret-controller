package com.example.demo.controller;

import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.ui.Model;
import org.springframework.web.bind.annotation.*;
import org.springframework.web.client.RestTemplate;
import org.springframework.beans.factory.annotation.Value;

@RestController
public class ExampleController {
    @Value("${secret1.USERNAME}")
    private String SECRET1_USERNAME_PATH;

    @Value("${secret1.PASSWORD}")
    private String SECRET1_PASSWORD_PATH;

    @Value("${secret2.USERNAME}")
    private String SECRET2_USERNAME_PATH;

    @Value("${secret2.PASSWORD}")
    private String SECRET2_PASSWORD_PATH;

    @Value("${SECRET1_USERNAME_ENV}")
    private String SECRET1_USERNAME_ENV;

    @Value("${SECRET1_PASSWORD_ENV}")
    private String SECRET1_PASSWORD_ENV;

    @Value("${SECRET2_USERNAME_ENV}")
    private String SECRET2_USERNAME_ENV;

    @Value("${SECRET2_PASSWORD_ENV}")
    private String SECRET2_PASSWORD_ENV;

    @GetMapping("/log-secret")
    public String getSecrets() {
        System.out.println("=================================");
        System.out.println("MOUNTED FILE SECRET");
        System.out.println("USERNAME1: " + SECRET1_USERNAME_PATH);
        System.out.println("PASSWORD1: " + SECRET1_PASSWORD_PATH);
        System.out.println("USERNAME2: " + SECRET2_USERNAME_PATH);
        System.out.println("PASSWORD2: " + SECRET2_PASSWORD_PATH);
        System.out.println();
        System.out.println("ENVIRONMENT VARIABLES SECRET");
        System.out.println("USERNAME1: " + SECRET1_USERNAME_ENV);
        System.out.println("PASSWORD1: " + SECRET1_PASSWORD_ENV);
        System.out.println("USERNAME2: " + SECRET2_USERNAME_ENV);
        System.out.println("PASSWORD2: " + SECRET2_PASSWORD_ENV);
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
