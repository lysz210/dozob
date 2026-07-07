plugins {
    kotlin("jvm") version "2.3.21"
    `java-library`
    `maven-publish`
    id("com.google.protobuf") version "0.10.0"
}

group = "it.lysz210.akasha"
version = "0.0.1"

val protobufVersion = project.property("protobufVersion")
val grpcVersion = "1.82.1"
val quarkusGrpcVersion = "3.37.1"

repositories {
    mavenCentral()
    mavenLocal()
    gradlePluginPortal()
}

dependencies {
    api("com.google.protobuf:protobuf-kotlin:${protobufVersion}")
    api("io.grpc:grpc-stub:${grpcVersion}")
    api("io.grpc:grpc-protobuf:${grpcVersion}")
    api("jakarta.annotation:jakarta.annotation-api:2.1.1")

    api("io.smallrye.reactive:mutiny:2.6.0")
    api("io.quarkus:quarkus-grpc:${quarkusGrpcVersion}")
}

java {
    sourceCompatibility = JavaVersion.VERSION_25
    targetCompatibility = JavaVersion.VERSION_25
}

kotlin {
    compilerOptions {
        jvmTarget = org.jetbrains.kotlin.gradle.dsl.JvmTarget.JVM_25
        javaParameters = true
    }
}

sourceSets {
    main {
        proto {
            // Point this to the root folder where your proto files are located
            srcDir("pkg/storage")
        }
    }
}

protobuf {
    // Configure the protoc executable
    protoc {
        artifact = "com.google.protobuf:protoc:${protobufVersion}"
    }

    plugins {
        create("grpc") {
            artifact = "io.grpc:protoc-gen-grpc-java:${grpcVersion}" // match your grpc version
        }
        create("mutiny") {
            artifact = "io.quarkus:quarkus-grpc-protoc-plugin:${quarkusGrpcVersion}:shaded@jar"
        }
    }


    // Generates Kotlin extensions in addition to Java code
    generateProtoTasks {
        all().forEach { task ->
            task.builtins {
                create("kotlin")
            }
            task.plugins {
                create("grpc")
                create("mutiny") {
                    option("generate-interfaces=true")
                    option("generate-clients=false")
                    option("generate-beans=false")
                }
            }
        }
    }
}

publishing {
    publications {
        create<MavenPublication>("mavenJava") {
            from(components["java"])

            groupId = project.group.toString()
            artifactId = project.name
            version = project.version.toString()
        }
    }
    repositories {
        maven {
            name = "GitHubPackages"
            // Replace with your GitHub Username/Org and Repo Name
            url = uri("https://maven.pkg.github.com/${System.getenv("GITHUB_REPOSITORY")}")
            credentials {
                username = System.getenv("GITHUB_ACTOR")
                password = System.getenv("GITHUB_TOKEN")
            }
        }
    }
}