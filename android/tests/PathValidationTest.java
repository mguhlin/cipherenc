import java.lang.reflect.Method;

/** Exercises the actual bridge path validator without starting Android UI. */
public final class PathValidationTest {
    public static void main(String[] arguments) throws Exception {
        Class<?> activity = Class.forName("org.mguhlin.cipherenc.MainActivity");
        Method validator = activity.getDeclaredMethod("safePath", String.class);
        validator.setAccessible(true);
        for (String valid : new String[]{"note.txt", "Folder/notes [draft].txt", "Español/archivo.cen"}) {
            if (!(Boolean)validator.invoke(null, valid)) throw new AssertionError("Valid path rejected: " + valid);
        }
        for (String invalid : new String[]{"", "/absolute", "../escape", "folder/../escape", "folder//file", "folder/", "bad\\file", "bad:file", "bad\nfile", "bad\u0000file"}) {
            if ((Boolean)validator.invoke(null, invalid)) throw new AssertionError("Unsafe path accepted");
        }
        System.out.println("Android bridge path traversal/control-character checks PASS.");
    }
}
