package com.example.careold.controller;

import com.example.careold.common.DefineData;
import com.example.careold.common.PhotoUtil;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.core.io.FileSystemResource;
import org.springframework.core.io.Resource;
import org.springframework.http.MediaType;
import org.springframework.http.ResponseEntity;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.PostMapping;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RequestParam;
import org.springframework.web.bind.annotation.RestController;
import org.springframework.web.multipart.MultipartFile;
import java.io.IOException;
import java.nio.file.Files;
import java.nio.file.Path;
import java.nio.file.Paths;
import java.util.Map;

@RestController
@RequestMapping("/restful/image")
public class ImagesApi {
    @Value("${local-path}")
    private String path;

    @PostMapping("/saveImage")
    public ResponseEntity<Map<String, String>> saveImage(@RequestParam("file") MultipartFile file) {
        Map<String, String> saved = PhotoUtil.saveFile(file, path);
        if (!DefineData.PHOTO_SAVE_SUCCESS.equals(saved.get(DefineData.ERROR_CODE))) {
            return ResponseEntity.badRequest().body(Map.of("returnCode", "0000", "message", saved.get(DefineData.ERROR_MESSAGE)));
        }
        return ResponseEntity.ok(Map.of("returnCode", "1111", "filename", saved.get("filename")));
    }

    @GetMapping("/download")
    public ResponseEntity<Resource> download(@RequestParam("name") String filename) throws IOException {
        return image(path, filename);
    }

    private ResponseEntity<Resource> image(String base, String name) throws IOException {
        Path file = resolveDownloadPath(base, name);
        if (file == null) return ResponseEntity.notFound().build();
        String extension = name.substring(name.lastIndexOf('.') + 1).toLowerCase(java.util.Locale.ROOT);
        MediaType type = extension.equals("png") ? MediaType.IMAGE_PNG : extension.equals("bmp") ? MediaType.parseMediaType("image/bmp") : MediaType.IMAGE_JPEG;
        return ResponseEntity.ok().contentType(type).contentLength(Files.size(file))
                .header("Cache-Control", "private, no-store").body(new FileSystemResource(file));
    }

    private static Path resolveDownloadPath(String base, String name) throws IOException {
        if (base == null || name == null || name.contains("/") || name.contains("\\") || name.contains(":")
                || !name.toLowerCase(java.util.Locale.ROOT).matches("[^\\x00]+\\.(png|jpe?g|bmp)")) return null;
        Path directory = Paths.get(base);
        if (!Files.isDirectory(directory)) return null;
        Path root = directory.toRealPath();
        Path candidate;
        try { candidate = root.resolve(name).normalize(); }
        catch (java.nio.file.InvalidPathException e) { return null; }
        if (!candidate.startsWith(root) || !Files.isRegularFile(candidate)) return null;
        Path real = candidate.toRealPath();
        return real.startsWith(root) ? real : null;
    }
}
